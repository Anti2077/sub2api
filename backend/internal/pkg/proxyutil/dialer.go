// Package proxyutil 提供统一的代理配置功能
//
// 支持的代理协议：
//   - HTTP/HTTPS: 通过 Transport.Proxy 设置
//   - SOCKS5: 通过 Transport.DialContext 设置（客户端本地解析 DNS）
//   - SOCKS5H: 通过 Transport.DialContext 设置（代理端远程解析 DNS，推荐）
//   - Hysteria 2: 通过 Transport.DialContext 设置（QUIC/TLS 隧道）
//
// 注意：proxyurl.Parse() 会自动将 socks5:// 升级为 socks5h://，
// 确保 DNS 也由代理端解析，防止 DNS 泄漏。
package proxyutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	hysteriaclient "github.com/apernet/hysteria/core/v2/client"
	"golang.org/x/net/proxy"
)

const (
	// socks5DialTimeout 限制到 SOCKS5 代理自身的 TCP 建连耗时。
	socks5DialTimeout = 10 * time.Second
	// socks5DialKeepAlive 与 Go 默认 keepalive 探测间隔保持一致。
	socks5DialKeepAlive = 30 * time.Second
)

// socks5ForwardDialer 是 SOCKS5 dialer 的底层拨号器。
//
// proxy.FromURL 的默认 forward dialer 是 proxy.Direct（零值 net.Dialer，无超时），
// 代理地址不可达时会一直卡到内核 TCP 重传耗尽（Linux 约 130 秒）。SOCKS5 分支会
// 覆盖 Transport.DialContext，因此调用方在 Transport 上设置的建连超时对这条路径
// 无效，必须在这里补上。
var socks5ForwardDialer = &net.Dialer{
	Timeout:   socks5DialTimeout,
	KeepAlive: socks5DialKeepAlive,
}

// ConfigureTransportProxy 根据代理 URL 配置 Transport
//
// 支持的协议：
//   - http/https: 设置 transport.Proxy
//   - socks5: 设置 transport.DialContext（客户端本地解析 DNS）
//   - socks5h: 设置 transport.DialContext（代理端远程解析 DNS，推荐）
//   - hysteria2: 设置 transport.DialContext（Hysteria 2 TCP 隧道）
//
// 参数：
//   - transport: 需要配置的 http.Transport
//   - proxyURL: 代理地址，nil 表示直连
//
// 返回：
//   - error: 代理配置错误（协议不支持或 dialer 创建失败）
func ConfigureTransportProxy(transport *http.Transport, proxyURL *url.URL) error {
	if proxyURL == nil {
		return nil
	}

	scheme := strings.ToLower(proxyURL.Scheme)
	switch scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)
		return nil

	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(proxyURL, socks5ForwardDialer)
		if err != nil {
			return fmt.Errorf("create socks5 dialer: %w", err)
		}
		// 优先使用支持 context 的 DialContext，以支持请求取消和超时
		if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = contextDialer.DialContext
		} else {
			// 回退路径：如果 dialer 不支持 ContextDialer，则包装为简单的 DialContext
			// 注意：此回退不支持请求取消和超时控制
			transport.DialContext = func(_ context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
		return nil

	case "hysteria2":
		dialContext, err := NewHysteria2DialContext(proxyURL)
		if err != nil {
			return err
		}
		transport.Proxy = nil
		transport.DialContext = dialContext
		return nil

	default:
		return fmt.Errorf("unsupported proxy scheme: %s", scheme)
	}
}

// NewHysteria2DialContext creates a context-aware TCP dialer backed by a
// Hysteria 2 client session. The target address is sent to the Hysteria 2
// server, so the target hostname is resolved remotely by the server.
func NewHysteria2DialContext(proxyURL *url.URL) (func(context.Context, string, string) (net.Conn, error), error) {
	if proxyURL == nil || !strings.EqualFold(proxyURL.Scheme, "hysteria2") {
		return nil, fmt.Errorf("hysteria2 proxy URL is required")
	}

	serverHost := proxyURL.Hostname()
	serverPort := proxyURL.Port()
	if serverHost == "" || serverPort == "" {
		return nil, fmt.Errorf("hysteria2 proxy URL requires host and port")
	}
	serverAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(serverHost, serverPort))
	if err != nil {
		return nil, fmt.Errorf("resolve hysteria2 server address: %w", err)
	}

	auth := ""
	if proxyURL.User != nil {
		auth = proxyURL.User.Username()
		if password, ok := proxyURL.User.Password(); ok {
			auth = password
		}
	}
	serverName := proxyURL.Query().Get("sni")
	if serverName == "" {
		serverName = serverHost
	}
	if insecure := strings.ToLower(proxyURL.Query().Get("insecure")); insecure == "1" || insecure == "true" || insecure == "yes" {
		return nil, fmt.Errorf("hysteria2 insecure TLS verification is not allowed")
	}

	client, err := hysteriaclient.NewReconnectableClient(func() (*hysteriaclient.Config, error) {
		return &hysteriaclient.Config{
			ServerAddr: serverAddr,
			Auth:       auth,
			TLSConfig: hysteriaclient.TLSConfig{
				ServerName: serverName,
			},
			QUICConfig: hysteriaclient.QUICConfig{
				MaxIdleTimeout: 10 * time.Second,
			},
		}, nil
	}, nil, true)
	if err != nil {
		return nil, fmt.Errorf("create hysteria2 client: %w", err)
	}
	d := &hysteria2Dialer{client: client}
	return d.DialContext, nil
}

type hysteria2Dialer struct {
	client hysteriaclient.Client
}

func (d *hysteria2Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("hysteria2 only supports TCP targets, got %s", network)
	}

	type dialResult struct {
		conn net.Conn
		err  error
	}
	result := make(chan dialResult, 1)
	go func() {
		conn, err := d.client.TCP(address)
		result <- dialResult{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		go func() {
			if result := <-result; result.conn != nil {
				_ = result.conn.Close()
			}
		}()
		return nil, ctx.Err()
	case result := <-result:
		if result.err != nil {
			return nil, fmt.Errorf("hysteria2 TCP dial: %w", result.err)
		}
		return result.conn, nil
	}
}

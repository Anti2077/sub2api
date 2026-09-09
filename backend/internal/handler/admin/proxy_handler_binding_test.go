package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProxyRequestBindingAcceptsHysteria2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		body   string
		target any
	}{
		{
			name:   "create",
			body:   `{"name":"hy2","protocol":"hysteria2","host":"proxy.example.com","port":443,"password":"secret"}`,
			target: &CreateProxyRequest{},
		},
		{
			name:   "update",
			body:   `{"protocol":"hysteria2"}`,
			target: &UpdateProxyRequest{},
		},
		{
			name:   "batch create",
			body:   `{"proxies":[{"protocol":"hysteria2","host":"proxy.example.com","port":443,"password":"secret"}]}`,
			target: &BatchCreateRequest{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tc.body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			require.NoError(t, ctx.ShouldBindJSON(tc.target))
		})
	}
}

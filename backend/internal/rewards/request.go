package rewards

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func FreeText(c *Capture) bool { return c != nil && c.Free && c.TextOnly }

// RequestIdentity inspects a bounded prefix and always restores the original body.
// Media, tools and unknown endpoints never qualify for zero-balance admission.
func RequestIdentity(r *http.Request) (string, bool, error) {
	path := r.URL.Path
	valid := strings.HasSuffix(path, "/messages") || strings.HasSuffix(path, "/responses") || strings.HasSuffix(path, "/chat/completions") || strings.HasSuffix(path, "/completions") || strings.HasSuffix(path, "/systemone") || strings.Contains(path, ":generateContent") || strings.Contains(path, ":streamGenerateContent")
	if !valid || r.Method != "POST" || r.Body == nil {
		return "", false, nil
	}
	body := r.Body
	b, e := io.ReadAll(io.LimitReader(body, 2*1024*1024+1))
	r.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(b), body), body}
	if e != nil {
		return "", false, e
	}
	if len(b) > 2*1024*1024 {
		return "", false, nil
	}
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		return "", false, nil
	}
	model, _ := v["model"].(string)
	if model == "" {
		if i := strings.Index(path, "/models/"); i >= 0 {
			model = strings.SplitN(path[i+8:], ":", 2)[0]
		}
	}
	return strings.TrimSpace(model), pureText(v), nil
}
func pureText(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, a := range x {
			switch strings.ToLower(k) {
			case "tools", "tool_choice", "functions", "modalities", "audio", "image", "images", "image_url", "file", "file_id", "file_data", "inline_data", "inlinedata", "filedata", "file_data_uri", "responsemodalities", "response_modalities", "web_search_options":
				return false
			case "type":
				if s, ok := a.(string); ok && s != "text" && s != "input_text" && s != "output_text" && s != "message" && s != "function_call_output" {
					return false
				}
			}
			if !pureText(a) {
				return false
			}
		}
	case []any:
		for _, a := range x {
			if !pureText(a) {
				return false
			}
		}
	}
	return true
}

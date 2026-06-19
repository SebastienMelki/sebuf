package tsclientgen

import (
	"net/http"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// TestRequestParamName verifies that the request parameter is named "req" only
// when the generated method reads it (body, path params, or GET/DELETE query
// params), and "_req" otherwise. This guards against the regression where an
// empty-field request on a POST was named "_req" while the body referenced
// "req".
func TestRequestParamName(t *testing.T) {
	oneQuery := make([]annotations.QueryParam, 1)

	tests := []struct {
		name string
		cfg  rpcMethodConfig
		want string
	}{
		{
			name: "POST with empty body still uses req",
			cfg:  rpcMethodConfig{httpMethod: http.MethodPost, hasBody: true},
			want: "req",
		},
		{
			name: "PUT with body uses req",
			cfg:  rpcMethodConfig{httpMethod: http.MethodPut, hasBody: true},
			want: "req",
		},
		{
			name: "GET with no params uses _req",
			cfg:  rpcMethodConfig{httpMethod: http.MethodGet},
			want: "_req",
		},
		{
			name: "GET with path param uses req",
			cfg:  rpcMethodConfig{httpMethod: http.MethodGet, pathParams: []string{"id"}},
			want: "req",
		},
		{
			name: "GET with query params uses req",
			cfg:  rpcMethodConfig{httpMethod: http.MethodGet, queryParams: oneQuery},
			want: "req",
		},
		{
			name: "DELETE with query params uses req",
			cfg:  rpcMethodConfig{httpMethod: http.MethodDelete, queryParams: oneQuery},
			want: "req",
		},
		{
			name: "non-GET/DELETE verb with query params but no body uses _req",
			cfg:  rpcMethodConfig{httpMethod: http.MethodHead, queryParams: oneQuery},
			want: "_req",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.requestParamName(); got != tt.want {
				t.Errorf("requestParamName() = %q, want %q", got, tt.want)
			}
		})
	}
}

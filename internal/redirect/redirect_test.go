package redirect_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/CruGlobal/mirage-server/internal/redirect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirect_ServeHTTP(t *testing.T) {
	type response struct {
		status  int
		headers http.Header
	}
	tests := []struct {
		name      string
		url       string
		redirect  redirect.Redirect
		response  response
		expectErr bool
	}{
		{
			name: "missing location",
			redirect: redirect.Redirect{
				Location: "",
			},
			expectErr: true,
		},
		{
			name: "defaults",
			redirect: redirect.Redirect{
				Location: "example.com",
			},
			response: response{
				status:  302,
				headers: map[string][]string{"Location": {"https://example.com"}},
			},
		},
		{
			name: "permanent redirect",
			redirect: redirect.Redirect{
				Location: "example.com",
				Status:   redirect.StatusPermanent,
			},
			response: response{
				status:  301,
				headers: map[string][]string{"Location": {"https://example.com"}},
			},
		},
		{
			name: "redirect with invalid rewrite",
			url:  "https://www.example.com/foo/bar",
			redirect: redirect.Redirect{
				Location: "example.com",
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: nil},
						Replace: "$1",
						Final:   true,
					},
				},
			},
			response: response{
				status:  302,
				headers: map[string][]string{"Location": {"https://example.com"}},
			},
		},
		{
			name: "redirect with rewrite",
			url:  "https://www.example.com/foo/bar",
			redirect: redirect.Redirect{
				Location: "example.com",
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^(.*)$`)},
						Replace: "$1",
						Final:   true,
					},
				},
			},
			response: response{
				status:  302,
				headers: map[string][]string{"Location": {"https://example.com/foo/bar"}},
			},
		},
		{
			name: "redirect with multiple rewrites",
			url:  "https://www.example.com/foo/bar",
			redirect: redirect.Redirect{
				Location: "example.com",
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^/(.*)$`)},
						Replace: "/prefix/$1",
						Final:   false,
					},
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`bar`)},
						Replace: "baz",
						Final:   true,
					},
				},
			},
			response: response{
				status:  302,
				headers: map[string][]string{"Location": {"https://example.com/prefix/foo/baz"}},
			},
		},
		{
			name: "redirect with multiple rewrites, matches second",
			url:  "https://www.example.com/foo/bar",
			redirect: redirect.Redirect{
				Location: "example.com",
				Rewrites: []redirect.Rewrite{
					{
						// Doesn't match the first rewrite
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^/hello/(.*)$`)},
						Replace: "$1",
						Final:   true,
					},
					{
						// The second rewrite should match
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`bar`)},
						Replace: "baz",
						Final:   true,
					},
				},
			},
			response: response{
				status:  302,
				headers: map[string][]string{"Location": {"https://example.com/foo/baz"}},
			},
		},
		{
			name: "redirect with multiple rewrites, first final",
			url:  "https://www.example.com/foo/bar",
			redirect: redirect.Redirect{
				Location: "example.com",
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^/foo(.*)$`)},
						Replace: "$1",
						Final:   true,
					},
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`bar`)},
						Replace: "baz",
						Final:   true,
					},
				},
			},
			response: response{
				status:  302,
				headers: map[string][]string{"Location": {"https://example.com/bar"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest(http.MethodGet, tt.url, nil)
			require.NoError(t, err)
			err = tt.redirect.ServeHTTP(w, r)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.response.status, w.Code)
			assert.Equal(t, tt.response.headers, w.Result().Header)
		})
	}
}

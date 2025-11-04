package redirect_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/CruGlobal/mirage-server/internal/redirect"
	"github.com/caddyserver/caddy/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirect_Prefix(t *testing.T) {
	tests := []struct {
		name      string
		redirect  redirect.Redirect
		url       string
		expect    map[string]any
		expectErr bool
	}{
		{
			name:      "defaults",
			url:       "https://www.example.com",
			expectErr: false,
			expect: map[string]any{
				"http.mirage.type":              redirect.TypeRedirect.String(),
				"http.mirage.redirect.location": "https://example.com",
				"http.mirage.redirect.status":   redirect.StatusTemporary.StatusCode(),
			},
			redirect: redirect.Redirect{
				Location: "example.com",
			},
		},
		{
			name:      "temporary redirect",
			url:       "https://www.example.com",
			expectErr: false,
			expect: map[string]any{
				"http.mirage.type":              redirect.TypeRedirect.String(),
				"http.mirage.redirect.location": "https://example.com",
				"http.mirage.redirect.status":   redirect.StatusTemporary.StatusCode(),
			},
			redirect: redirect.Redirect{
				Location: "example.com",
				Type:     redirect.TypeRedirect,
				Status:   redirect.StatusTemporary,
			},
		},
		{
			name:      "permanent redirect",
			url:       "https://www.example.com",
			expectErr: false,
			expect: map[string]any{
				"http.mirage.type":              redirect.TypeRedirect.String(),
				"http.mirage.redirect.location": "https://example.com",
				"http.mirage.redirect.status":   redirect.StatusPermanent.StatusCode(),
			},
			redirect: redirect.Redirect{
				Location: "example.com",
				Type:     redirect.TypeRedirect,
				Status:   redirect.StatusPermanent,
			},
		},
		{
			name:      "permanent redirect with rewrite",
			url:       "https://www.example.com/foo/bar/baz",
			expectErr: false,
			expect: map[string]any{
				"http.mirage.type":              redirect.TypeRedirect.String(),
				"http.mirage.redirect.location": "https://example.com/foo/bar/baz",
				"http.mirage.redirect.status":   redirect.StatusPermanent.StatusCode(),
			},
			redirect: redirect.Redirect{
				Location: "example.com",
				Type:     redirect.TypeRedirect,
				Status:   redirect.StatusPermanent,
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^(.*)$`)},
						Replace: "$1",
						Final:   false,
					},
				},
			},
		},
		{
			name: "basic proxy",
			url:  "https://www.example.com",
			expect: map[string]any{
				"http.mirage.type":           redirect.TypeProxy.String(),
				"http.mirage.proxy.upstream": "example.info:443",
				"http.mirage.proxy.path":     "",
			},
			redirect: redirect.Redirect{
				Location: "example.info",
				Type:     redirect.TypeProxy,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, tt.url, nil)
			require.NoError(t, err)

			repl := caddy.NewReplacer()

			err = tt.redirect.Process(r, repl)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			for k, v := range tt.expect {
				actual, exists := repl.Get(k)
				assert.True(t, exists)
				assert.Equal(t, v, actual)
			}
		})
	}
}

func TestRedirect_RewritePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		location string
		redirect redirect.Redirect
		expected string
	}{
		{
			name:     "empty path, no rewrite",
			path:     "",
			expected: "",
			redirect: redirect.Redirect{},
		},
		{
			name:     "empty path, no rewrite, has location",
			path:     "",
			location: "/lorem/ipsum",
			expected: "/lorem/ipsum",
			redirect: redirect.Redirect{},
		},
		{
			name:     "empty path, with rewrite",
			path:     "",
			expected: "",
			redirect: redirect.Redirect{
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^(.*)$`)},
						Replace: "$1",
						Final:   true,
					},
				},
			},
		},
		{
			name:     "empty path, with rewrite, with location",
			path:     "",
			location: "/lorem/ipsum",
			expected: "",
			redirect: redirect.Redirect{
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^(.*)$`)},
						Replace: "$1",
						Final:   true,
					},
				},
			},
		},
		{
			name:     "invalid rewrite",
			path:     "/foo/bar/baz",
			expected: "",
			redirect: redirect.Redirect{
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: nil},
						Replace: "$1",
						Final:   true,
					},
				},
			},
		},
		{
			name:     "forward path",
			path:     "/foo/bar/baz",
			location: "/lorem/ipsum",
			expected: "/foo/bar/baz",
			redirect: redirect.Redirect{
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^(.*)$`)},
						Replace: "$1",
						Final:   true,
					},
				},
			},
		},
		{
			name:     "forward path, multiple rewrites",
			path:     "/foo/bar/baz",
			location: "/lorem/ipsum",
			expected: "/foo/qux/baz",
			redirect: redirect.Redirect{
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^(.*)$`)},
						Replace: "$1",
						Final:   false,
					},
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`bar`)},
						Replace: "qux",
						Final:   true,
					},
				},
			},
		},
		{
			name:     "forward path, multiple rewrites, final rewrite",
			path:     "/foo/bar/baz",
			location: "/lorem/ipsum",
			expected: "/bar/baz",
			redirect: redirect.Redirect{
				Rewrites: []redirect.Rewrite{
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`^/foo(.*)$`)},
						Replace: "$1",
						Final:   true,
					},
					{
						RegExp:  redirect.RewriteRegexp{Regexp: regexp.MustCompile(`bar`)},
						Replace: "qux",
						Final:   true,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.redirect.RewritePath(tt.path, tt.location)
			assert.Equal(t, tt.expected, path)
		})
	}
}

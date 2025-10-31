package redirect

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
)

type Redirect struct {
	Hostname string    `dynamodbav:"Hostname"`
	Type     Type      `dynamodbav:"Type"`
	Location string    `dynamodbav:"Location"`
	Status   Status    `dynamodbav:"Status"`
	Rewrites []Rewrite `dynamodbav:"Rewrites"`
}

func (r *Redirect) Process(request *http.Request, repl *caddy.Replacer) error {
	repl.Set("http.mirage.type", r.Type.String())
	path := r.RewritePath(request.URL.Path)

	switch r.Type {
	case TypeRedirect:
		repl.Set("http.mirage.redirect.location", fmt.Sprintf("https://%s%s", r.Location, path))
		repl.Set("http.mirage.redirect.status", r.Status.StatusCode())
	case TypeProxy:
		return nil
	}

	return nil
}

func (r *Redirect) RewritePath(path string) string {
	hasMatch := false
	if r.Rewrites != nil {
		for _, rewrite := range r.Rewrites {
			if rewrite.RegExp.Regexp == nil {
				continue
			}
			if rewrite.RegExp.MatchString(path) {
				hasMatch = true
				path = rewrite.RegExp.ReplaceAllString(path, rewrite.Replace)
				if rewrite.Final {
					break
				}
			}
		}
	}
	if !hasMatch {
		return ""
	}
	return path
}

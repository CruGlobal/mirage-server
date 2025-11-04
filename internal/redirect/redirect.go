package redirect

import (
	"fmt"
	"net/http"
	"net/url"

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
	location, err := url.Parse(fmt.Sprintf("https://%s", r.Location))
	if err != nil {
		return err
	}
	repl.Set("http.mirage.type", r.Type.String())

	location.Path = r.RewritePath(request.URL.Path, location.Path)

	switch r.Type {
	case TypeRedirect:
		repl.Set("http.mirage.redirect.location", location.String())
		repl.Set("http.mirage.redirect.status", r.Status.StatusCode())
	case TypeProxy:
		repl.Set("http.mirage.proxy.upstream", fmt.Sprintf("https://%s", location.Host))
		repl.Set("http.mirage.proxy.path", location.Path)
		return nil
	}

	return nil
}

func (r *Redirect) RewritePath(requestPath string, locationPath string) string {
	hasMatch := false
	rewritePath := requestPath
	if r.Rewrites != nil {
		for _, rewrite := range r.Rewrites {
			if rewrite.RegExp.Regexp == nil {
				continue
			}
			if rewrite.RegExp.MatchString(rewritePath) {
				hasMatch = true
				rewritePath = rewrite.RegExp.ReplaceAllString(rewritePath, rewrite.Replace)
				if rewrite.Final {
					break
				}
			}
		}
	}
	if !hasMatch {
		return locationPath
	}
	return rewritePath
}

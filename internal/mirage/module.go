package mirage

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/CruGlobal/mirage-server/internal/app"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

var (
	// Interface guards.
	_ caddy.Provisioner           = (*Mirage)(nil)
	_ caddy.Module                = (*Mirage)(nil)
	_ caddyhttp.MiddlewareHandler = (*Mirage)(nil)
)

func init() {
	caddy.RegisterModule(Mirage{})
	httpcaddyfile.RegisterHandlerDirective("mirage", parseCaddyfile)
}

type Mirage struct {
	Table  string           `json:"table,omitempty"`
	Key    string           `json:"key,omitempty"`
	Client *dynamodb.Client `json:"-"`

	logger *zap.Logger
}

func NewMirage() *Mirage {
	return &Mirage{
		Table: app.DefaultTable,
		Key:   app.DefaultKey,
	}
}

func (r Mirage) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.mirage",
		New: func() caddy.Module {
			return NewMirage()
		},
	}
}

func (r *Mirage) Provision(ctx caddy.Context) error {
	r.logger = ctx.Logger(r)

	module, err := ctx.App(app.AppName)
	if err != nil {
		return err
	}

	m, ok := module.(*app.App)
	if !ok {
		return fmt.Errorf("unexpected module type: %T", module)
	}
	if m == nil {
		return errors.New("mirage has not been initialized")
	}

	if m.Client == nil {
		return errors.New("DynamoDB client has not been initialized")
	}

	r.Client = m.Client
	r.Table = m.Table
	r.Client = m.Client

	return nil
}

func parseCaddyfile(_ httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	// Mirage has no configuration, so we just return a new instance
	return NewMirage(), nil
}

func (r Mirage) ServeHTTP(writer http.ResponseWriter, request *http.Request, handler caddyhttp.Handler) error {
	// Split host and port
	hostname, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		hostname = request.Host // Probably OK, host just didn't have a port
	}
	useCache := !request.URL.Query().Has("skip_cache")

	writer.Header().Set("Server", "mirage")

	redirect, err := r.GetRedirect(request.Context(), hostname, useCache)
	if err != nil {
		return handler.ServeHTTP(writer, request)
	}
	return redirect.ServeHTTP(writer, request)
}

func (r *Mirage) GetRedirect(ctx context.Context, hostname string, _ bool) (*Redirect, error) {
	// TODO: Implement caching
	item, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			r.Key: &types.AttributeValueMemberS{Value: hostname},
		},
	})
	if err != nil {
		return nil, err
	}
	if item.Item == nil {
		return nil, errors.New("no redirect found")
	}

	var redirect Redirect
	err = attributevalue.UnmarshalMap(item.Item, &redirect)
	if err != nil {
		return nil, err
	}
	return &redirect, nil
}

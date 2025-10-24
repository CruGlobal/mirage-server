package miragetest

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/stretchr/testify/require"
	tcddb "github.com/testcontainers/testcontainers-go/modules/dynamodb"
)

type TestConfig struct {
	Region   string
	Endpoint string
	Table    string
	Key      string
}

func NewMirageCaddyContext(t *testing.T, config TestConfig) caddy.Context {
	t.Helper()

	caddyfileInput := fmt.Sprintf(`{
	mirage {
		region %s
		endpoint %s
		table %s
		key %s
	}
	log {
		level ERROR
	}
}
`,
		config.Region,
		config.Endpoint,
		config.Table,
		config.Key,
	)
	adapter := caddyfile.Adapter{ServerType: &httpcaddyfile.ServerType{}}
	adaptedJSON, warnings, err := adapter.Adapt([]byte(caddyfileInput), nil)
	require.NoError(t, err)
	require.Empty(t, warnings)

	cfg := &caddy.Config{}
	err = caddy.StrictUnmarshalJSON(adaptedJSON, cfg)
	require.NoError(t, err)

	ctx, err := caddy.ProvisionContext(cfg)
	require.NoError(t, err)

	return ctx
}

func CreateDynamoDBContainer(t *testing.T) (string, *tcddb.DynamoDBContainer) {
	t.Helper()
	ddbc, err := tcddb.Run(t.Context(), "amazon/dynamodb-local:latest", tcddb.WithDisableTelemetry())
	require.NoError(t, err)

	var endpoint string
	endpoint, err = ddbc.ConnectionString(t.Context())
	require.NoError(t, err)
	return fmt.Sprintf("http://%s", endpoint), ddbc
}

func CreateDynamoDBTable(t *testing.T, client *dynamodb.Client, table string, key string) {
	t.Helper()
	t.Logf("create dynamodb table %s with key %s", table, key)
	_, err := client.CreateTable(t.Context(), &dynamodb.CreateTableInput{
		TableName:   aws.String(table),
		BillingMode: types.BillingModePayPerRequest,
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String(key),
				KeyType:       types.KeyTypeHash,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String(key),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
	})
	require.NoError(t, err)
}

func DeleteDynamoDBTable(t *testing.T, client *dynamodb.Client, table string) {
	t.Helper()
	t.Logf("delete dynamodb table %s", table)
	_, err := client.DeleteTable(t.Context(), &dynamodb.DeleteTableInput{
		TableName: aws.String(table),
	})
	require.NoError(t, err)
}

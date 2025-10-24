package miragetest

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/stretchr/testify/require"
)

func NewMirageCaddyContext(t *testing.T) caddy.Context {
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
		os.Getenv("DYNAMODB_TESTING_REGION"),
		os.Getenv("DYNAMODB_TESTING_ENDPOINT"),
		os.Getenv("DYNAMODB_TESTING_TABLE"),
		os.Getenv("DYNAMODB_TESTING_KEY"),
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

func CreateDynamoDBTable(t *testing.T, client *dynamodb.Client, table string, key string) {
	t.Helper()
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
	_, err := client.DeleteTable(t.Context(), &dynamodb.DeleteTableInput{
		TableName: aws.String(table),
	})
	require.NoError(t, err)
}

package permission_test

import (
	"os"
	"testing"

	"github.com/CruGlobal/mirage-server/internal/app"
	"github.com/CruGlobal/mirage-server/internal/permission"
	"github.com/CruGlobal/mirage-server/miragetest"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestPermission_NewPermission(t *testing.T) {
	perm := permission.NewPermission()
	assert.NotNil(t, perm)
	assert.IsType(t, &permission.Permission{}, perm)
	assert.Equal(t, app.DefaultTable, perm.Table)
	assert.Equal(t, app.DefaultKey, perm.Key)
}

func TestPermission_CaddyModule(t *testing.T) {
	module := permission.Permission{}.CaddyModule()
	assert.IsType(t, caddy.ModuleInfo{}, module)
	assert.Equal(t, caddy.ModuleID("tls.permission.dynamodb"), module.ID)
	assert.IsType(t, &permission.Permission{}, module.New())
}

func TestPermission_UnmarshalCaddyfile(t *testing.T) {
	testcases := []struct {
		name      string
		caddyfile string
		expectErr bool
	}{
		{
			name:      "valid1",
			caddyfile: `dynamodb`,
			expectErr: false,
		},
		{
			name:      "invalid1",
			caddyfile: `dynamodb {}`,
			expectErr: true,
		},
		{
			name: "invalid2",
			caddyfile: `dynamodb {
			}`,
			expectErr: true,
		},
		{
			name: "invalid3",
			caddyfile: `dynamodb name {
			}`,
			expectErr: true,
		},
		{
			name: "invalid4",
			caddyfile: `dynamodb {
				key TestKey
			}`,
			expectErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := permission.NewPermission()
			err := p.UnmarshalCaddyfile(caddyfile.NewTestDispenser(tc.caddyfile))
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPermission_Provision(t *testing.T) {
	ctx := miragetest.NewMirageCaddyContext(t)

	perm := permission.NewPermission()
	err := perm.Provision(ctx)
	require.NoError(t, err)

	assert.NotNil(t, perm.Client)
	assert.Equal(t, os.Getenv("DYNAMODB_TESTING_TABLE"), perm.Table)
	assert.Equal(t, os.Getenv("DYNAMODB_TESTING_KEY"), perm.Key)
}

type PermissionTestSuite struct {
	suite.Suite

	permission *permission.Permission
}

func (ts *PermissionTestSuite) SetupSuite() {
	ctx := miragetest.NewMirageCaddyContext(ts.T())

	perm := permission.NewPermission()
	err := perm.Provision(ctx)
	ts.Require().NoError(err)

	miragetest.CreateDynamoDBTable(ts.T(), perm.Client, perm.Table, perm.Key)

	ts.permission = perm
}

func (ts *PermissionTestSuite) TearDownSuite() {
	miragetest.DeleteDynamoDBTable(ts.T(), ts.permission.Client, ts.permission.Table)
}

func (ts *PermissionTestSuite) TestPermission_CertificateAllowed() {
	ctx := ts.T().Context()
	validKeys := []string{"example.com", "www.example.com", "starkindustries.com"}
	invalidKeys := []string{"www.starkindustries.com", "ftp.example.com"}

	for _, key := range validKeys {
		_, err := ts.permission.Client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(ts.permission.Table),
			Item: map[string]types.AttributeValue{
				ts.permission.Key: &types.AttributeValueMemberS{Value: key},
			},
		})
		ts.Require().NoError(err)
	}

	for _, valid := range validKeys {
		err := ts.permission.CertificateAllowed(ctx, valid)
		ts.Require().NoError(err)
	}

	for _, invalid := range invalidKeys {
		err := ts.permission.CertificateAllowed(ctx, invalid)
		ts.Require().Error(err)
	}
}

func TestPermissionTestSuite(t *testing.T) {
	suite.Run(t, new(PermissionTestSuite))
}

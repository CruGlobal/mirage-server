package permission_test

import (
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
	tcddb "github.com/testcontainers/testcontainers-go/modules/dynamodb"
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
	ctx := miragetest.NewMirageCaddyContext(t, miragetest.TestConfig{
		Region:   "us-east-1",
		Endpoint: "http://example.com:8000",
		Table:    "MirageServerConfigTest",
		Key:      "Hostname",
	})

	perm := permission.NewPermission()
	err := perm.Provision(ctx)
	require.NoError(t, err)

	assert.NotNil(t, perm.Client)
	assert.Equal(t, "MirageServerConfigTest", perm.Table)
	assert.Equal(t, "Hostname", perm.Key)
}

type PermissionTestSuite struct {
	suite.Suite

	permission *permission.Permission
	ddbc       *tcddb.DynamoDBContainer
}

func (ts *PermissionTestSuite) SetupSuite() {
	endpoint, ddbc := miragetest.CreateDynamoDBContainer(ts.T())
	ts.ddbc = ddbc

	ctx := miragetest.NewMirageCaddyContext(ts.T(), miragetest.TestConfig{
		Region:   "us-east-1",
		Endpoint: endpoint,
		Table:    "MirageServerConfigTest",
		Key:      "Hostname",
	})

	perm := permission.NewPermission()
	err := perm.Provision(ctx)
	ts.Require().NoError(err)
	ts.permission = perm
}

func (ts *PermissionTestSuite) TearDownSuite() {
	if ts.ddbc != nil {
		err := ts.ddbc.Terminate(ts.T().Context())
		ts.Require().NoError(err)
	}
}

func (ts *PermissionTestSuite) SetupTest() {
	miragetest.CreateDynamoDBTable(ts.T(), ts.permission.Client, ts.permission.Table, ts.permission.Key)
}

func (ts *PermissionTestSuite) TearDownTest() {
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

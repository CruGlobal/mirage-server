package mirage_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/CruGlobal/mirage-server/internal/app"
	"github.com/CruGlobal/mirage-server/internal/mirage"
	"github.com/CruGlobal/mirage-server/miragetest"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/caddyserver/caddy/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestMirage_NewMirage(t *testing.T) {
	r := mirage.NewMirage()
	assert.NotNil(t, r)
	assert.IsType(t, &mirage.Mirage{}, r)
	assert.Equal(t, app.DefaultTable, r.Table)
	assert.Equal(t, app.DefaultKey, r.Key)
}

func TestMirage_CaddyModule(t *testing.T) {
	module := mirage.Mirage{}.CaddyModule()
	assert.IsType(t, caddy.ModuleInfo{}, module)
	assert.Equal(t, caddy.ModuleID("http.handlers.mirage"), module.ID)
	assert.IsType(t, &mirage.Mirage{}, module.New())
}

func TestMirage_Provision(t *testing.T) {
	ctx := miragetest.NewMirageCaddyContext(t)

	r := mirage.NewMirage()
	err := r.Provision(ctx)
	require.NoError(t, err)

	assert.NotNil(t, r.Client)
	assert.Equal(t, os.Getenv("DYNAMODB_TESTING_TABLE"), r.Table)
	assert.Equal(t, os.Getenv("DYNAMODB_TESTING_KEY"), r.Key)
}

type MirageTestSuite struct {
	suite.Suite

	mirage *mirage.Mirage
}

func (ts *MirageTestSuite) SetupSuite() {
	ctx := miragetest.NewMirageCaddyContext(ts.T())

	r := mirage.NewMirage()
	err := r.Provision(ctx)
	ts.Require().NoError(err)

	ts.mirage = r
}

//nolint:gochecknoglobals // redirects are global to the test suite
var redirects = []mirage.Redirect{
	{
		Hostname: "www.example.com",
		Location: "example.com",
	},
	{
		Hostname: "example.org",
		Type:     mirage.TypeRedirect,
		Status:   mirage.StatusPermanent,
		Location: "www.example.org",
	},
	{
		Hostname: "www.example.info",
		Type:     mirage.TypeRedirect,
		Status:   mirage.StatusTemporary,
		Location: "example.info",
		Rewrites: []mirage.Rewrite{
			{
				RegExp:  mirage.RewriteRegexp{Regexp: regexp.MustCompile(`^(.*)$`)},
				Replace: "$1",
				Final:   true,
			},
		},
	},
}

// SetupTest creates the table before each test.
func (ts *MirageTestSuite) SetupTest() {
	miragetest.CreateDynamoDBTable(ts.T(), ts.mirage.Client, ts.mirage.Table, ts.mirage.Key)

	for _, r := range redirects {
		item, err := attributevalue.MarshalMap(r)
		ts.Require().NoError(err)
		_, err = ts.mirage.Client.PutItem(ts.T().Context(), &dynamodb.PutItemInput{
			TableName: aws.String(ts.mirage.Table),
			Item:      item,
		})
		ts.Require().NoError(err)
	}
}

// TearDownTest deletes the table after each test.
func (ts *MirageTestSuite) TearDownTest() {
	miragetest.DeleteDynamoDBTable(ts.T(), ts.mirage.Client, ts.mirage.Table)
}

// TestMirageTestSuite runs the test suite.
func TestMirageTestSuite(t *testing.T) {
	suite.Run(t, new(MirageTestSuite))
}

func (ts *MirageTestSuite) TestMirage_GetRedirect() {
	tests := []struct {
		name      string
		hostname  string
		expect    mirage.Redirect
		expectErr bool
	}{
		{
			name:     "temporary redirect",
			hostname: "www.example.com",
			expect:   redirects[0],
		},
		{
			name:     "permanent redirect",
			hostname: "example.org",
			expect:   redirects[1],
		},
		{
			name:      "missing redirect",
			hostname:  "example.edu",
			expectErr: true,
		},
		{
			name:     "redirect with rewrites",
			hostname: "www.example.info",
			expect:   redirects[2],
		},
	}
	for _, tc := range tests {
		ts.Run(tc.name, func() {
			r, err := ts.mirage.GetRedirect(ts.T().Context(), tc.hostname, true)
			if tc.expectErr {
				ts.Require().Error(err)
				return
			}
			ts.Require().NoError(err)
			ts.Equal(tc.expect, *r)
		})
	}
}

type MockCaddyHandler struct {
	mock.Mock
}

func (m *MockCaddyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) error {
	args := m.Called(writer, request)
	return args.Error(0)
}

func (ts *MirageTestSuite) TestMirage_ServeHTTP() {
	type response struct {
		status  int
		headers http.Header
	}
	tests := []struct {
		name      string
		method    string
		url       string
		response  response
		expectErr bool
	}{
		{
			name:   "temporary redirect",
			method: "GET",
			url:    "https://www.example.com",
			response: response{
				status:  302,
				headers: map[string][]string{"Location": {"https://example.com"}, "Server": {"mirage"}},
			},
		},
		{
			name:   "permanent redirect",
			method: "GET",
			url:    "https://example.org",
			response: response{
				status:  301,
				headers: map[string][]string{"Location": {"https://www.example.org"}, "Server": {"mirage"}},
			},
		},
		{
			name:      "unknown hostname",
			method:    "GET",
			url:       "https://example.edu",
			expectErr: true,
		},
		{
			name:   "redirect with rewrites",
			method: "GET",
			url:    "https://www.example.info/foo/bar/baz",
			response: response{
				status: 302,
				headers: map[string][]string{
					"Location": {"https://example.info/foo/bar/baz"},
					"Server":   {"mirage"},
				},
			},
		},
	}
	for _, tc := range tests {
		ts.Run(tc.name, func() {
			w := httptest.NewRecorder()
			r, err := http.NewRequest(tc.method, tc.url, nil)
			ts.Require().NoError(err)
			mockHandler := new(MockCaddyHandler)
			mockHandler.On("ServeHTTP", mock.Anything, mock.Anything).Return(nil)

			err = ts.mirage.ServeHTTP(w, r, mockHandler)
			ts.Require().NoError(err)
			if tc.expectErr {
				mockHandler.AssertNumberOfCalls(ts.T(), "ServeHTTP", 1)
			} else {
				mockHandler.AssertNotCalled(ts.T(), "ServeHTTP")
				ts.Equal(tc.response.status, w.Code)
				ts.Equal(tc.response.headers, w.Result().Header)
			}
		})
	}
}

package cache_test

import (
	"testing"

	"github.com/CruGlobal/mirage-server/internal/cache"
	"github.com/CruGlobal/mirage-server/internal/redirect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectCache_NewRedirectCache(t *testing.T) {
	c := cache.NewRedirectCache()
	assert.IsType(t, &cache.RedirectCache{}, c)
}

func TestRedirectCache_CRUD(t *testing.T) {
	c := cache.NewRedirectCache()
	assert.NotNil(t, c)

	example := redirect.Redirect{
		Hostname: "www.example.com",
		Type:     redirect.TypeRedirect,
		Status:   redirect.StatusPermanent,
		Location: "example.com",
	}

	c.Start()
	var redir redirect.Redirect
	err := c.Get("www.example.com", &redir)
	require.Error(t, err)

	c.Set(example)
	err = c.Get("www.example.com", &redir)
	require.NoError(t, err)
	assert.Equal(t, example, redir)

	c.Delete("www.example.com")
	err = c.Get("www.example.com", &redir)
	require.Error(t, err)

	c.Stop()
}

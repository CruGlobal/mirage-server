package cache

import (
	"fmt"

	"github.com/CruGlobal/mirage-server/internal/redirect"
	"github.com/jellydator/ttlcache/v3"
)

const (
	DefaultCapacity = 10000
)

type RedirectCache struct {
	cache *ttlcache.Cache[string, redirect.Redirect]
}

func NewRedirectCache() *RedirectCache {
	c := &RedirectCache{
		cache: ttlcache.New[string, redirect.Redirect](
			ttlcache.WithCapacity[string, redirect.Redirect](DefaultCapacity),
		),
	}
	return c
}

func (rc *RedirectCache) Start() {
	go rc.cache.Start()
}

func (rc *RedirectCache) Stop() {
	rc.cache.Stop()
}

func (rc *RedirectCache) Set(redirect redirect.Redirect) {
	rc.cache.Set(redirect.Hostname, redirect, ttlcache.DefaultTTL)
}

func (rc *RedirectCache) Get(hostname string, redirect *redirect.Redirect) error {
	item := rc.cache.Get(hostname)
	if item != nil {
		*redirect = item.Value()
		return nil
	}
	return fmt.Errorf("redirect not found for hostname: %s", hostname)
}

func (rc *RedirectCache) Delete(hostname string) {
	rc.cache.Delete(hostname)
}

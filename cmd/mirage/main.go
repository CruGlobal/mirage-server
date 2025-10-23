package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	_ "github.com/CruGlobal/mirage-server/internal/app"
	_ "github.com/CruGlobal/mirage-server/internal/mirage"
	_ "github.com/CruGlobal/mirage-server/internal/permission"
	_ "github.com/CruGlobal/mirage-server/internal/storage"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
)

func main() {
	caddycmd.Main()
}

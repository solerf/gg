package main

import (
	"github.com/alecthomas/kong"
)

// dlv debug --headless --api-version=2 --listen=127.0.0.1:43000 . -- repos -u user

func main() {
	kong.UsageOnError()
	ctx := kong.Parse(cli, kong.Description(description))
	ctx.FatalIfErrorf(ctx.Run())
}

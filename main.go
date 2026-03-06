package main

import (
	"github.com/alecthomas/kong"
)

func main() {
	kong.UsageOnError()
	ctx := kong.Parse(cli, kong.Description(description))
	ctx.FatalIfErrorf(ctx.Run())
}

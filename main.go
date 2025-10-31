package main

import (
	"github.com/alecthomas/kong"
)

func main() {
	kong.UsageOnError()
	k := kong.Parse(cli, kong.Description(description))
	k.FatalIfErrorf(k.Run())
}

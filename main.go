package main

import (
	"github.com/alecthomas/kong"
)

func main() {
	/*f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()*/
	kong.UsageOnError()
	k := kong.Parse(cli, kong.Description(description))
	k.FatalIfErrorf(k.Run(cli.Debug))
}

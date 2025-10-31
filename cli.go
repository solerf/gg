package main

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/solerf/gg/github"
)

type gg struct {
	User    string `arg:"" required:"" help:"GitHub username"`
	HomeDir string `optional:"" type:"path" short:"d" default:"$HOME" env:"HOME" help:"$HOME directory"`
}

var description = "List GitHub user repositories"
var cli = &gg{}

func (g *gg) Run() error {
	client, err := github.Client(g.HomeDir, g.User)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repos, err := client.Repos(ctx, client.Config.User)
	if err != nil {
		return err
	}

	tmplFile := "_output.md.tmpl"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		return err
	}

	var bbuffer bytes.Buffer
	err = tmpl.Execute(&bbuffer, repos)
	if err != nil {
		return err
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(120),
		glamour.WithInlineTableLinks(true),
	)
	if err != nil {
		return err
	}

	out, err := renderer.Render(bbuffer.String())
	if err != nil {
		return err
	}
	fmt.Print(out)

	return nil
}

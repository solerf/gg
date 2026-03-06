package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/solerf/gg/github"
	"github.com/solerf/gg/tui"
)

type CloneCmd struct {
	User    string `arg:"" required:"" help:"GitHub username"`
	HomeDir string `optional:"" type:"path" short:"d" default:"$HOME" env:"HOME" help:"$HOME directory"`
}

type CreateCmd struct {
	User           string `arg:"" required:"" help:"GitHub username"`
	RepositoryName string `required:"" short:"r" help:"GitHub new repository name"`
	Visibility     string `option:"" optional:"" short:"v" default:"public" help:"GitHub new repository visibility"`
	HomeDir        string `optional:"" type:"path" short:"d" default:"$HOME" env:"HOME" help:"$HOME directory"`
}

type gg struct {
	Clone  CloneCmd  `cmd:"" help:"select repository to be cloned"`
	Create CreateCmd `cmd:"" help:"create a new repository at remote"`
	// TODO add field to inform PTA if not from the default path
	// PtaFile string `arg:"" optional:"" help:"Absolute path to user's PTA"`
	// TODO properly implement debug
	Debug bool `optional:"" long:"debug" default:"false" help:"debug logs to file"`
}

var description = "List GitHub user repositories"
var cli = &gg{}

func (g *CloneCmd) Run(debug bool) error {
	curDir, err := os.Getwd()
	if err != nil {
		return err
	}

	client, err := github.NewClient(g.HomeDir, g.User)
	if err != nil {
		return err
	}

	model, err := tui.NewModel(debug, curDir, g.HomeDir, g.User, client)
	if err != nil {
		return err
	}

	p := tea.NewProgram(model)
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return err
	}

	switch tm := m.(type) {
	case tui.Model:
		if tm.Done {
			// keep the result on stdout
			fmt.Println(m.View())
		}
	}

	return nil
}

func (g *CreateCmd) Run(debug bool) error {
	client, err := github.NewClient(g.HomeDir, g.User)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	repository, err := client.CreateRepository(ctx, g.RepositoryName, strings.ToLower(g.Visibility) != "public")
	if err != nil {
		return err
	}

	fmt.Printf("Created repository %s\n", repository.FullName)
	return nil
}

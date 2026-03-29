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

type ReposCmd struct {
}

type CreateCmd struct {
	RepositoryName string `required:"" short:"r" help:"GitHub new repository name"`
	Visibility     string `option:"" optional:"" short:"v" default:"public" help:"GitHub new repository visibility"`
}

type gg struct {
	Repos  ReposCmd  `cmd:"" help:"list repositories and allow cloning"`
	Create CreateCmd `cmd:"" help:"create a new repository at remote"`

	HomeDir string `optional:"" type:"path" short:"d" default:"$HOME" env:"HOME" help:"$HOME directory"`
	User    string `required:"" type:"string" short:"u" help:"GitHub username"`

	PtaPath string `optional:"" type:"path" help:"Absolute path to user's PTA location"`
}

var description = "Details from GitHub user repositories"
var cli = &gg{}

func (cl *ReposCmd) Run(gg *gg) error {
	// the gg is auto injected
	curDir, err := os.Getwd()
	if err != nil {
		return err
	}

	client, err := github.NewClient(gg.HomeDir, gg.PtaPath, gg.User)
	if err != nil {
		return err
	}

	model, err := tui.NewModel(curDir, client)
	if err != nil {
		return err
	}

	if _, err = tea.NewProgram(model).Run(); err != nil {
		return fmt.Errorf("running tui: %w", err)
	}
	return nil
}

func (cr *CreateCmd) Run(gg *gg) error {
	// the gg is auto injected
	client, err := github.NewClient(gg.HomeDir, gg.PtaPath, gg.User)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	repository, err := client.CreateRepository(ctx, cr.RepositoryName, strings.ToLower(cr.Visibility) != "public")
	if err != nil {
		return err
	}

	fmt.Printf("Created repository %s\n", repository.FullName)
	return nil
}

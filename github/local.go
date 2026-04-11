package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func Clone(ctx context.Context, repository *Repository, destination string) error {
	return run(ctx, "clone", repository.CloneUrl, destination)
}

func run(ctx context.Context, args ...string) error {
	git, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git not found: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, git, args...)
	cmd.Stdout = nil
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		command := strings.Join(append([]string{git}, args...), " ")
		return fmt.Errorf("%v: %w %v", command, err, stderr.String())
	}
	return nil
}

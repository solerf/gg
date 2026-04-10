package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func Clone(repository *Repository, destination string) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelFunc()
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

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
	return run(ctx, "git", "clone", repository.CloneUrl, destination)
}

func run(ctx context.Context, name string, args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, fmt.Sprintf(`/usr/bin/%s`, name), args...)
	cmd.Stdout = nil
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		command := strings.Join(append([]string{name}, args...), " ")
		return fmt.Errorf("%v: %w %v", command, err, stderr.String())
	}
	return nil
}

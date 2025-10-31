package github

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	ptaExtension = ".pta"
	configFile   = ".ggconf"
)

type config struct {
	User    string `msgpack:"user"`
	PathPTA string `msgpack:"path_pta"`
	PTA     string `msgpack:"-"`
}

func newConfig(homeDir, gitUser string) (*config, error) {
	conf, err := readConfig(homeDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if conf != nil {
		return conf, nil
	}

	pta, bytes, err := getPTA(homeDir, gitUser)
	if err != nil {
		return nil, err
	}

	c, err := writeConfig(homeDir, gitUser, pta, bytes)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func readConfig(homeDir string) (*config, error) {
	c, err := os.ReadFile(filepath.Join(homeDir, configFile))
	if err != nil {
		return nil, err
	}

	var conf *config
	if err = msgpack.Unmarshal(c, &conf); err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(conf.PathPTA)
	if err != nil {
		return nil, err
	}

	conf.PTA = strings.TrimSpace(string(bytes))
	return conf, nil
}

func writeConfig(homeDir string, user string, pathPTA string, pta []byte) (*config, error) {
	conf := &config{User: user, PathPTA: pathPTA, PTA: strings.TrimSpace(string(pta))}
	marshal, err := msgpack.Marshal(conf)
	if err != nil {
		return nil, err
	}

	if err = os.WriteFile(filepath.Join(homeDir, configFile), marshal, 0600); err != nil {
		return nil, err
	}
	return conf, nil
}

func getPTA(homeDir, gitUser string) (string, []byte, error) {
	var ptaPath string
	target := fmt.Sprintf("%s%s", gitUser, ptaExtension)
	err := filepath.WalkDir(homeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && d.Name() == target {
			ptaPath = path
			return io.EOF
		}

		return nil
	})

	if err != nil && !errors.Is(err, io.EOF) {
		return "", nil, err
	}

	bytes, err := os.ReadFile(ptaPath)
	if err != nil {
		return "", nil, err
	}
	return ptaPath, bytes, nil
}

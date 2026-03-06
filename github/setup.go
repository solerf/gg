package github

import (
	"errors"
	"fmt"
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
	if err != nil {
		return nil, err
	}

	if conf != nil {
		return conf, nil
	}

	ptaDir := fmt.Sprintf("%v/.config/git/pta/%v.pta", homeDir, gitUser)
	pta, bytes, err := readPTA(ptaDir, gitUser)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read pta: %w", err)
	}

	c, err := writeConfig(homeDir, gitUser, pta, bytes)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func readConfig(homeDir string) (*config, error) {
	configPath := filepath.Join(homeDir, configFile)
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	c, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// TODO deserialize this as a map with user=pta
	var conf *config
	if err = msgpack.Unmarshal(c, &conf); err != nil {
		return nil, fmt.Errorf("unmarshall config: %w", err)
	}

	bytes, err := os.ReadFile(conf.PathPTA)
	if err != nil {
		return nil, fmt.Errorf("read config path: %w", err)
	}

	conf.PTA = strings.TrimSpace(string(bytes))
	return conf, nil
}

func writeConfig(homeDir string, user string, pathPTA string, pta []byte) (*config, error) {
	// TODO serialize this as a map with user=pta
	conf := &config{User: user, PathPTA: pathPTA, PTA: strings.TrimSpace(string(pta))}
	marshal, err := msgpack.Marshal(conf)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	if err = os.WriteFile(filepath.Join(homeDir, configFile), marshal, 0600); err != nil {
		return nil, fmt.Errorf("writing config: %w", err)
	}
	return conf, nil
}

func readPTA(targetDir, gitUser string) (string, []byte, error) {
	target := fmt.Sprintf("%s%s", gitUser, ptaExtension)

	var ptaPath string
	err := filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir %v: %w", path, err)
		}

		if !d.IsDir() && d.Name() == target {
			ptaPath = path
			return nil
		}

		return nil
	})

	if len(ptaPath) == 0 || err != nil {
		return "", nil, fmt.Errorf("pta `%v`: %w", ptaPath, os.ErrNotExist)
	}

	bytes, err := os.ReadFile(ptaPath)
	if err != nil {
		return "", nil, fmt.Errorf("read pta config: %w", err)
	}
	return ptaPath, bytes, nil
}

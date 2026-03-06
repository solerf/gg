package github

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	configFile = ".ggconf"
)

type config struct {
	User    string `msgpack:"user"`
	PtaPath string `msgpack:"pta_path"`
	Pta     string `msgpack:"-"`
}

func newConfig(homeDir, ptaPath, gitUser string) (*config, error) {
	configs, err := load(homeDir)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	if len(configs) > 0 {
		if c, exists := configs[gitUser]; exists {
			pta, err := loadPta(c.PtaPath)
			if err != nil {
				return nil, fmt.Errorf("config stored: %w", err)
			}

			if len(pta) > 0 {
				c.Pta = pta
				return &c, nil
			}
		}
	}

	if len(ptaPath) > 0 {
		pta, err := loadPta(fmt.Sprintf("%s/%s.pat", ptaPath, gitUser))
		if err != nil {
			return nil, fmt.Errorf("config from PTA path: %w", err)
		}

		if len(pta) > 0 {
			// just ignore the return
			_ = upsert(homeDir, gitUser, ptaPath)
			return &config{
				User:    gitUser,
				PtaPath: ptaPath,
				Pta:     pta,
			}, nil
		}
	}

	defaultPtaDir := fmt.Sprintf("%v/.config/git/pta/%v.pta", homeDir, gitUser)
	pta, err := loadPta(defaultPtaDir)
	if err != nil {
		return nil, fmt.Errorf("config impossible to set PTA: %w", err)
	}

	if len(pta) == 0 {
		return nil, errors.New("config impossible to set PTA, nothing found")
	}

	// just ignore the return
	_ = upsert(homeDir, gitUser, defaultPtaDir)
	return &config{
		User:    gitUser,
		PtaPath: defaultPtaDir,
		Pta:     pta,
	}, nil
}

func loadPta(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("stat pta: %w", err)
	}

	ptaBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("load pta: %w", err)
	}

	return strings.TrimSpace(string(ptaBytes)), nil
}

func load(homeDir string) (map[string]config, error) {
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

	var configs map[string]config
	if err = msgpack.Unmarshal(c, &configs); err != nil {
		return nil, fmt.Errorf("unmarshall config: %w", err)
	}
	return configs, nil
}

func upsert(homeDir, user, ptaPath string) error {
	loaded, err := load(homeDir)
	if err != nil {
		return err
	}

	if loaded == nil {
		loaded = make(map[string]config, 1)
	}

	loaded[user] = config{User: user, PtaPath: ptaPath}

	marshal, err := msgpack.Marshal(loaded)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err = os.WriteFile(filepath.Join(homeDir, configFile), marshal, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

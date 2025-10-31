package github

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	repositoriesUrl = "https://api.github.com/search/repositories?q=user:%v"
)

type GitHub struct {
	Config *config
	httpc  *http.Client
}

func Client(homeDir, gitUser string) (*GitHub, error) {
	c, err := newConfig(homeDir, gitUser)
	if err != nil {
		return nil, err
	}

	defTransport := http.DefaultTransport.(*http.Transport)
	defTransport.IdleConnTimeout = time.Minute * 5
	defTransport.MaxConnsPerHost = 10
	defTransport.MaxIdleConns = 5
	defTransport.MaxIdleConnsPerHost = 5
	defHttpC := &http.Client{Transport: defTransport}
	return &GitHub{Config: c, httpc: defHttpC}, nil
}

func (gh *GitHub) Repos(ctx context.Context, gitUser string) ([]Repository, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(repositoriesUrl, gitUser), http.NoBody)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Accept", "application/vnd.github.v3+json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", gh.Config.PTA))

	response, err := gh.httpc.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("bad response status: %v", response.Status)
	}

	repos, err := toModel(response.Body)
	if err != nil {
		return nil, err
	}
	return repos, nil
}

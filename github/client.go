package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	searchRepositoriesUrl = "https://api.github.com/search/repositories?q=user:%v"
	createRepositoriesUrl = "https://api.github.com/user/repos"
)

type Client struct {
	Config *config
	httpc  *http.Client
}

func NewClient(homeDir, gitUser string) (*Client, error) {
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
	return &Client{Config: c, httpc: defHttpC}, nil
}

func (gh *Client) ListRepositories(ctx context.Context, gitUser string) ([]Repository, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(searchRepositoriesUrl, gitUser), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("list repositories request: %w", err)
	}

	responseBody, err := gh.doRequest(request)
	if err != nil {
		return nil, fmt.Errorf("list repositories request: %w", err)
	}
	defer responseBody.Close()

	var ghr ghRepositories
	if err = json.NewDecoder(responseBody).Decode(&ghr); err != nil {
		return nil, fmt.Errorf("list repositories read response: %w", err)
	}
	return ghRepositoriesResponseToModel(ghr), nil
}

func (gh *Client) CreateRepository(ctx context.Context, name string, private bool) (*Repository, error) {
	body, err := json.Marshal(map[string]interface{}{
		"name":    name,
		"private": private,
	})
	if err != nil {
		return nil, fmt.Errorf("create repository params: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, createRepositoriesUrl, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create repository request: %w", err)
	}

	responseBody, err := gh.doRequest(request)
	if err != nil {
		return nil, fmt.Errorf("create repository call: %w", err)
	}
	defer responseBody.Close()

	var ghr ghRepository
	if err = json.NewDecoder(responseBody).Decode(&ghr); err != nil {
		return nil, fmt.Errorf("create repository read response: %w", err)
	}
	repository := ghRepositoryToModel(ghr)
	return &repository, nil
}

func (gh *Client) doRequest(request *http.Request) (io.ReadCloser, error) {
	request.Header.Add("Accept", "application/vnd.github.v3+json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", gh.Config.PTA))

	response, err := gh.httpc.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("bad response status: %v", response.Status)
	}
	return response.Body, nil
}

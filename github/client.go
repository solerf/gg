package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	var ghr ghRepositoriesResponse
	if err = json.NewDecoder(response.Body).Decode(&ghr); err != nil {
		return nil, err
	}
	return ghRepositoriesResponseToModel(ghr), nil
}

func (gh *Client) CreateRepository(ctx context.Context, name string, private bool) (*Repository, error) {
	params := map[string]interface{}{
		"name":    name,
		"private": private,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, createRepositoriesUrl, bytes.NewReader(body))
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

	var ghr ghRepository
	if err = json.NewDecoder(response.Body).Decode(&ghr); err != nil {
		return nil, err
	}
	repository := ghRepositoryToModel(ghr)
	return &repository, nil
}

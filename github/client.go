package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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

func NewClient(homeDir, ptaPath, gitUser string) (*Client, error) {
	c, err := newConfig(homeDir, ptaPath, gitUser)
	if err != nil {
		return nil, fmt.Errorf("gh client: %w", err)
	}

	defTransport := http.DefaultTransport.(*http.Transport)
	defTransport.IdleConnTimeout = time.Minute * 5
	defTransport.MaxConnsPerHost = 10
	defTransport.MaxIdleConns = 5
	defTransport.MaxIdleConnsPerHost = 5
	defHttpC := &http.Client{Transport: defTransport}
	return &Client{Config: c, httpc: defHttpC}, nil
}

func (gh *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	var nextLinkRegex = regexp.MustCompile("<(.+)>; rel=\"next\".+")

	extractNextLink := func(headerLink string) string {
		// <https://api.github.com/search/repositories?q=user%3Asolerf&page=2>; rel="next", <https://api.github.com/search/repositories?q=user%3Asolerf&page=2>; rel="last"
		if len(headerLink) > 0 {
			submatch := nextLinkRegex.FindStringSubmatch(headerLink)
			if len(submatch) > 1 {
				return submatch[1]
			}
		}
		return ""
	}

	makeRequest := func(url string) ([]Repository, string, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			return nil, "", fmt.Errorf("gh client list repositories request: %w", err)
		}

		response, err := gh.doRequest(request)
		if err != nil {
			return nil, "", fmt.Errorf("gh client list repositories: %w", err)
		}

		var buff bytes.Buffer
		_, err = io.Copy(&buff, response.Body)
		if err != nil {
			return nil, "", fmt.Errorf("gh client repository read response: %w", err)
		}
		defer response.Body.Close()

		var ghr ghRepositories
		if err = json.NewDecoder(&buff).Decode(&ghr); err != nil {
			return nil, "", fmt.Errorf("gh client list repositories decoder: %w", err)
		}

		nextPage := extractNextLink(response.Header.Get("Link"))
		return ghRepositoriesResponseToModel(ghr), nextPage, nil
	}

	repos, nextPage, err := makeRequest(fmt.Sprintf(searchRepositoriesUrl, gh.Config.User))
	if err != nil {
		return nil, err
	}
	result := append([]Repository{}, repos...)

	for len(nextPage) != 0 {
		repos, nextPage, err = makeRequest(nextPage)
		if err != nil {
			return nil, err
		}
		result = append(result, repos...)
	}

	return result, nil
}

func (gh *Client) CreateRepository(ctx context.Context, name string, private bool) (*Repository, error) {
	body, err := json.Marshal(map[string]interface{}{
		"name":    name,
		"private": private,
	})
	if err != nil {
		return nil, fmt.Errorf("gh client create repository params: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, createRepositoriesUrl, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gh client create repository request: %w", err)
	}

	response, err := gh.doRequest(request)
	if err != nil {
		return nil, fmt.Errorf("gh client create repository: %w", err)
	}

	var buff bytes.Buffer
	_, err = io.Copy(&buff, response.Body)
	if err != nil {
		return nil, fmt.Errorf("gh client repository read response: %w", err)
	}
	defer response.Body.Close()

	var ghr ghRepository
	if err = json.NewDecoder(&buff).Decode(&ghr); err != nil {
		return nil, fmt.Errorf("gh client create repository decoder: %w", err)
	}
	repository := ghRepositoryToModel(ghr)
	return &repository, nil
}

func (gh *Client) doRequest(request *http.Request) (*http.Response, error) {
	request.Header.Add("Accept", "application/vnd.github.v3+json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", gh.Config.Pta))

	response, err := gh.httpc.Do(request)
	if err != nil {
		return nil, fmt.Errorf("processing request: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("bad response status: %v", response.Status)
	}
	return response, nil
}

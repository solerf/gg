package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const (
	userRepositoriesUrl = "https://api.github.com/user/repos"
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
		request.URL.Query().Add("per_page", "50")
		if err != nil {
			return nil, "", fmt.Errorf("gh client list repositories request: %w", err)
		}

		ghr, headerLink, err := doRequest[[]ghRepository](gh.httpc, request, gh.Config.Pta)
		if err != nil {
			return nil, "", fmt.Errorf("gh client list repositories: %w", err)
		}

		nextPage := extractNextLink(headerLink)
		return ghRepositoriesResponseToModel(ghr), nextPage, nil
	}

	repos, nextPage, err := makeRequest(userRepositoriesUrl)
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

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, userRepositoriesUrl, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gh client create repository request: %w", err)
	}

	ghr, _, err := doRequest[ghRepository](gh.httpc, request, gh.Config.Pta)
	if err != nil {
		return nil, fmt.Errorf("gh client create repository: %w", err)
	}

	return new(ghRepositoryToModel(ghr)), nil
}

func doRequest[T ghRepository | []ghRepository](httpc *http.Client, request *http.Request, pta string) (T, string, error) {
	request.Header.Add("Accept", "application/vnd.github.v3+json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", pta))

	var t T

	response, err := httpc.Do(request)
	if err != nil {
		return t, "", fmt.Errorf("processing request: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return t, "", fmt.Errorf("bad response status: %v", response.Status)
	}
	defer response.Body.Close()

	if err = json.NewDecoder(response.Body).Decode(&t); err != nil {
		return t, "", fmt.Errorf("processing response: %w", err)
	}
	return t, response.Header.Get("Link"), nil
}

package github

import (
	"time"
)

type Repository struct {
	User          string    `json:"user"`
	FullName      string    `json:"fullname"`
	Name          string    `json:"name"`
	Visibility    string    `json:"visibility"`
	DefaultBranch string    `json:"default_branch"`
	Lang          string    `json:"lang"`
	CloneUrl      string    `json:"clone_url"`
	HtmlUrl       string    `json:"html_url"`
	LastUpdate    time.Time `json:"last_update"`
}

type ghRepository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Private       bool        `json:"private"`
	HtmlUrl       string      `json:"html_url"`
	SshUrl        string      `json:"ssh_url"`
	Language      interface{} `json:"language"`
	DefaultBranch string      `json:"default_branch"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

func ghRepositoriesResponseToModel(ghRepos []ghRepository) []Repository {
	repos := make([]Repository, 0, len(ghRepos))
	for i := 0; i < len(ghRepos); i++ {
		repos = append(repos, ghRepositoryToModel(ghRepos[i]))
	}
	return repos
}

func ghRepositoryToModel(ghRepo ghRepository) Repository {
	var lang string
	// it can explode but is faster
	if ghRepo.Language != nil {
		lang = ghRepo.Language.(string)
	}

	visibility := "public"
	if ghRepo.Private {
		visibility = "private"
	}

	return Repository{
		User:          ghRepo.Owner.Login,
		FullName:      ghRepo.FullName,
		Name:          ghRepo.Name,
		Visibility:    visibility,
		DefaultBranch: ghRepo.DefaultBranch,
		Lang:          lang,
		CloneUrl:      ghRepo.SshUrl,
		HtmlUrl:       ghRepo.HtmlUrl,
		LastUpdate:    ghRepo.UpdatedAt,
	}
}

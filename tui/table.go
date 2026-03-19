package tui

import (
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/solerf/gg/github"
)

type sortField = string

const (
	sortNameField sortField = "N"
	sortLangField sortField = "L"
	sortDateField sortField = "U"
)

var repositoryTableCols = []table.Column{
	{Title: "[N]Name"},
	{Title: "[L]Lang"},
	{Title: "[U]Last Update"},
}

type repositoryTable struct {
	rows         []table.Row
	repositories []github.Repository
}

func (rt *repositoryTable) SortInPlace(by sortField) {
	var sortF func(a, b github.Repository) int
	switch by {
	case sortNameField:
		sortF = func(a, b github.Repository) int {
			return strings.Compare(a.Name, b.Name)
		}
	case sortLangField:
		sortF = func(a, b github.Repository) int {
			return strings.Compare(a.Lang, b.Lang)
		}
	case sortDateField:
		sortF = func(a, b github.Repository) int {
			if a.LastUpdate.Before(b.LastUpdate) {
				return 1
			}
			if a.LastUpdate.After(b.LastUpdate) {
				return -1
			}
			return 0
		}
	}
	slices.SortFunc(rt.repositories, sortF)
	rt.refreshRows()
}

func (rt *repositoryTable) refreshRows() {
	if len(rt.rows) == 0 {
		rt.rows = make([]table.Row, len(rt.repositories))
		for i := 0; i < len(repositoryTableCols); i++ {
			rt.rows[i] = make(table.Row, len(repositoryTableCols))
		}
	}

	fieldByIndex := func(index int, field int) string {
		switch field {
		case 0:
			return rt.repositories[index].Name
		case 1:
			return rt.repositories[index].Lang
		case 2:
			// TODO change the date formatting?
			return rt.repositories[index].LastUpdate.Format(time.RFC1123)
		}
		return ""
	}

	for i := 0; i < len(rt.rows); i++ {
		for j := 0; j < len(rt.rows[i]); j++ {
			rt.rows[i][j] = fieldByIndex(i, j)
		}
	}
}

func newRepositoriesTable(repositories []github.Repository) *repositoryTable {
	rows := make([]table.Row, len(repositories))
	for i := 0; i < len(repositories); i++ {
		rows[i] = []string{
			repositories[i].Name,
			repositories[i].Lang,
			repositories[i].LastUpdate.Format(time.RFC1123),
		}
	}

	r := &repositoryTable{
		rows:         rows,
		repositories: repositories,
	}
	r.SortInPlace(sortDateField)
	return r
}

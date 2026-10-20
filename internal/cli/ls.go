package cli

import (
	"context"
	"slices"
	"time"

	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/ui"
)

type lsCmd struct{}

func (lsCmd) Run(ctx context.Context, s *session) error {
	list, err := s.client.Assignments(ctx)
	if err != nil {
		return err
	}

	var rows [][]string
	for _, a := range slices.SortedStableFunc(slices.Values(list), byDeadline) {
		rows = append(rows, []string{deadline(a.End), a.Kind, a.Title, a.Course})
	}
	ui.PrintTable(rows)
	return nil
}

func byDeadline(a, b manaba.Assignment) int {
	return a.End.Compare(b.End)
}

func deadline(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("01/02 15:04")
}

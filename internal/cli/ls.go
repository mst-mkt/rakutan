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

	now := time.Now()
	var rows [][]string
	for _, a := range slices.SortedStableFunc(slices.Values(list), byDeadline) {
		rows = append(rows, []string{ui.Deadline(a.End, now), a.Kind, a.Title, a.Course})
	}
	ui.PrintTable(rows)
	return nil
}

func byDeadline(a, b manaba.Assignment) int {
	return a.End.Compare(b.End)
}

package cli

import (
	"context"
	"fmt"
	"os"
	"slices"
	"text/tabwriter"
	"time"

	"github.com/mst-mkt/rakutan/internal/manaba"
)

type lsCmd struct{}

func (lsCmd) Run(ctx context.Context, s *session) error {
	list, err := s.client.Assignments(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, a := range slices.SortedStableFunc(slices.Values(list), byDeadline) {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", deadline(a.End), a.Kind, a.Title, a.Course)
	}
	return w.Flush()
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

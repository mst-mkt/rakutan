package cli

import (
	"context"
	"fmt"

	"github.com/mst-mkt/rakutan/internal/archive"
	"github.com/mst-mkt/rakutan/internal/manaba"
)

type logCmd struct {
	Quiz string `arg:"" help:"小テストの URL (course_X_query_Y を含むもの)"`
}

func (c logCmd) Run(ctx context.Context, s *session) error {
	ref, ok := manaba.QuizRef(c.Quiz)
	if !ok {
		return fmt.Errorf("小テストの URL ではありません: %s", c.Quiz)
	}

	r, err := s.client.Review(ctx, ref)
	if err != nil {
		return err
	}
	file, err := saveLog(ctx, s, r)
	if err != nil {
		return err
	}

	fmt.Println(file)
	return nil
}

func saveLog(ctx context.Context, s *session, r *manaba.Review) (string, error) {
	return archive.Save(ctx, s.client, archiveDir, r)
}

package manaba

import (
	"context"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	jst       = time.FixedZone("JST", 9*60*60)
	quizRefRe = regexp.MustCompile(`course_\d+_query_\d+`)
)

type Assignment struct {
	Kind   string    `json:"kind"`
	Title  string    `json:"title"`
	Href   string    `json:"href"`
	Course string    `json:"course"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end,omitzero"`
}

func (c *Client) Assignments(ctx context.Context) ([]Assignment, error) {
	return parseWith(ParseAssignments)(c.Get(ctx, "home_library_query"))
}

func ParseAssignments(p *Page) ([]Assignment, error) {
	rows := p.Doc.Find("table.stdlist tr").FilterFunction(func(_ int, tr *goquery.Selection) bool {
		return tr.Find("td").Length() >= 5
	})
	return goquery.Map(rows, func(_ int, tr *goquery.Selection) Assignment {
		tds := tr.Find("td")
		link := tr.Find(".myassignments-title a")
		return Assignment{
			Kind:   text(tds.Eq(0)),
			Title:  text(link),
			Href:   link.AttrOr("href", ""),
			Course: text(tr.Find(".mycourse-title")),
			Start:  parseTime(text(tds.Eq(3))),
			End:    parseTime(text(tds.Eq(4))),
		}
	}), nil
}

func QuizRef(s string) (string, bool) {
	ref := quizRefRe.FindString(s)
	return ref, ref != ""
}

func parseTime(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, jst)
	return t
}

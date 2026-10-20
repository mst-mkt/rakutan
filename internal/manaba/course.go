package manaba

import (
	"context"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Course struct {
	Title   string `json:"title"`
	Href    string `json:"href"`
	Year    string `json:"year"`
	Term    string `json:"term"`
	Teacher string `json:"teacher"`
	Pending bool   `json:"pending"`
}

type Quiz struct {
	Title  string    `json:"title"`
	Href   string    `json:"href"`
	Status string    `json:"status"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}

func (q Quiz) Answerable() bool {
	_, isQuiz := QuizRef(q.Href)
	return isQuiz && strings.Contains(q.Status, "受付中") && !strings.Contains(q.Status, "提出済み")
}

func (c *Client) Courses(ctx context.Context) ([]Course, error) {
	return parseWith(ParseCourses)(c.Get(ctx, "home_course"))
}

func (c *Client) Quizzes(ctx context.Context, course Course) ([]Quiz, error) {
	return parseWith(ParseQuizzes)(c.Get(ctx, course.Href+"_query"))
}

func ParseCourses(p *Page) ([]Course, error) {
	rows := p.Doc.Find("table.courselist tr").FilterFunction(func(_ int, tr *goquery.Selection) bool {
		return tr.Find(".courselist-title a").Length() > 0
	})
	return goquery.Map(rows, func(_ int, tr *goquery.Selection) Course {
		tds := tr.Find("td")
		link := tr.Find(".courselist-title a")
		return Course{
			Title:   text(link),
			Href:    link.AttrOr("href", ""),
			Year:    text(tds.Eq(1)),
			Term:    text(tds.Eq(2)),
			Teacher: text(tds.Eq(3)),
			Pending: tr.Find(`img[src*="coursedeadline-on"]`).Length() > 0,
		}
	}), nil
}

func ParseQuizzes(p *Page) ([]Quiz, error) {
	rows := p.Doc.Find("table.stdlist tr").FilterFunction(func(_ int, tr *goquery.Selection) bool {
		return tr.Find("td.query-title a").Length() > 0
	})
	return goquery.Map(rows, func(_ int, tr *goquery.Selection) Quiz {
		tds := tr.Find("td")
		link := tr.Find("td.query-title a")
		return Quiz{
			Title:  text(link),
			Href:   link.AttrOr("href", ""),
			Status: text(tds.Eq(1)),
			Start:  parseTime(text(tds.Eq(2))),
			End:    parseTime(text(tds.Eq(3))),
		}
	}), nil
}

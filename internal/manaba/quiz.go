package manaba

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	startButton   = Button{"action_QueryStudent_querystart", "スタート"}
	saveButton    = Button{"action_QueryStudent_queryshow_quittostart", "中断（一時保存）"}
	confirmButton = Button{"action_QueryStudent_queryshow_confirm", "提出確認"}
	submitButton  = Button{"action_QueryStudent_querydone", "提出"}
	backButton    = Button{"action_QueryStudent_querystart", "戻る"}

	submittedAtRe = regexp.MustCompile(`提出日時：([^）)]+)`)

	ErrNotStartable = errors.New("manaba: quiz cannot be started")
	ErrNotSubmitted = errors.New("manaba: quiz has not been submitted")
)

type Cover struct {
	URL       string `json:"url"`
	Course    string `json:"course"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Submitted bool   `json:"submitted"`
	start     *Form
}

type Sheet struct {
	Course    string     `json:"course"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
	form      Form
}

type Confirmation struct {
	Questions []Question `json:"questions"`
	form      Form
}

type Review struct {
	URL         string     `json:"url"`
	Course      string     `json:"course"`
	Title       string     `json:"title"`
	SubmittedAt string     `json:"submitted_at"`
	Score       string     `json:"score"`
	Questions   []Question `json:"questions"`
	answerKey   string
}

func (c *Client) Cover(ctx context.Context, ref string) (*Cover, error) {
	return parseWith(ParseCover)(c.Get(ctx, ref))
}

func (c *Client) Start(ctx context.Context, cv *Cover) (*Sheet, error) {
	if cv.start == nil {
		return nil, ErrNotStartable
	}
	return parseWith(ParseSheet)(c.Submit(ctx, cv.start.Press(startButton)))
}

func (c *Client) Save(ctx context.Context, s *Sheet, ans Answers) error {
	_, err := c.Submit(ctx, s.fill(ans, saveButton))
	return err
}

func (c *Client) Confirm(ctx context.Context, s *Sheet, ans Answers) (*Confirmation, error) {
	return parseWith(ParseConfirmation)(c.Submit(ctx, s.fill(ans, confirmButton)))
}

func (c *Client) Back(ctx context.Context, cf *Confirmation) (*Sheet, error) {
	return parseWith(ParseSheet)(c.Submit(ctx, cf.form.Press(backButton)))
}

func (c *Client) Finish(ctx context.Context, cf *Confirmation) (*Review, error) {
	return parseWith(c.parseReview(ctx))(c.Submit(ctx, cf.form.Press(submitButton)))
}

func (c *Client) Review(ctx context.Context, ref string) (*Review, error) {
	return parseWith(c.parseReview(ctx))(c.Get(ctx, ref+"?ReviewMode=1"))
}

func (c *Client) parseReview(ctx context.Context) func(*Page) (*Review, error) {
	return func(p *Page) (*Review, error) {
		r, err := ParseReview(p)
		if err != nil || r.answerKey == "" {
			return r, err
		}

		key, err := parseWith(ParseAnswerKey)(c.Get(ctx, r.answerKey))
		if err != nil {
			return nil, err
		}
		return r.withAnswerKey(key), nil
	}
}

func (r *Review) withAnswerKey(key []Question) *Review {
	byID := make(map[string]Question, len(key))
	for _, k := range key {
		byID[k.ID] = k
	}

	out := *r
	out.Questions = make([]Question, len(r.Questions))
	for i, q := range r.Questions {
		q.Correct, q.Points = byID[q.ID].Answer, byID[q.ID].Score
		out.Questions[i] = q
	}
	return &out
}

func (cv *Cover) Startable() bool {
	return cv.start != nil
}

func (s *Sheet) fill(ans Answers, b Button) Form {
	values := url.Values{}
	for _, q := range s.Questions {
		if vs := q.formValues(ans[q.ID]); len(vs) > 0 {
			values[q.ID] = vs
		}
	}

	return s.form.With(values).Press(b)
}

func ParseCover(p *Page) (*Cover, error) {
	table := p.Doc.Find("table.stdlist-query")
	if table.Length() == 0 {
		return nil, fmt.Errorf("manaba: quiz cover not found on %s", p.URL)
	}

	status := text(table.Find("th").FilterFunction(func(_ int, th *goquery.Selection) bool {
		return text(th) == "状態"
	}).Next())
	cv := &Cover{
		URL:       p.URLWithoutQuery(),
		Course:    text(p.Doc.Find("#coursename")),
		Title:     title(p),
		Status:    status,
		Submitted: strings.Contains(status, "提出済み"),
	}

	if f, err := findForm(p, startButton.selector()); err == nil {
		cv.start = &f
	}
	return cv, nil
}

func ParseSheet(p *Page) (*Sheet, error) {
	f, err := findForm(p, `input[name="TargetQIDs"]`)
	if err != nil {
		return nil, err
	}
	return &Sheet{Course: text(p.Doc.Find("#coursename")), Title: title(p), Questions: parseQuestions(p), form: f}, nil
}

func ParseConfirmation(p *Page) (*Confirmation, error) {
	f, err := findForm(p, submitButton.selector())
	if err != nil {
		return nil, err
	}
	return &Confirmation{Questions: parseQuestions(p), form: f}, nil
}

func ParseReview(p *Page) (*Review, error) {
	if p.Doc.Find(".querysubmitted .querypaper").Length() == 0 {
		return nil, ErrNotSubmitted
	}
	return &Review{
		URL:         p.URLWithoutQuery(),
		Course:      text(p.Doc.Find("#coursename")),
		Title:       title(p),
		SubmittedAt: submatch(submittedAtRe, text(p.Doc.Find(".querysubmitter"))),
		Score:       strings.TrimSpace(strings.TrimPrefix(text(p.Doc.Find(".query-score")), "合計点 :")),
		Questions:   parseQuestions(p),
		answerKey:   answerKeyRef(p),
	}, nil
}

func ParseAnswerKey(p *Page) ([]Question, error) {
	if p.Doc.Find(".queryanswer .querypaper").Length() == 0 {
		return nil, fmt.Errorf("manaba: answer key not found on %s", p.URL)
	}
	return parseQuestions(p), nil
}

func answerKeyRef(p *Page) string {
	href, ok := p.Doc.Find(`.queryanswerspan a[href*="answer=1"]`).Attr("href")
	if !ok {
		return ""
	}
	return p.Abs(href)
}

func title(p *Page) string {
	return cmp.Or(
		text(p.Doc.Find("table.stdlist-query th").First()),
		text(p.Doc.Find(".queryframe tr.title th").First()),
		text(p.Doc.Find(".querypaper h1").First()),
	)
}

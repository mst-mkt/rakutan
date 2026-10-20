package manaba

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Kind string

const (
	KindText     Kind = "text"
	KindTextarea Kind = "textarea"
	KindRadio    Kind = "radio"
	KindCheckbox Kind = "checkbox"
	KindSelect   Kind = "select"
)

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Question struct {
	ID      string   `json:"id"`
	Kind    Kind     `json:"kind"`
	Text    string   `json:"text"`
	Images  []string `json:"images,omitempty"`
	Options []Option `json:"options,omitempty"`
	Answer  []string `json:"answer,omitempty"`
	Mark    string   `json:"mark,omitempty"`
	Score   string   `json:"score,omitempty"`
	Correct []string `json:"correct,omitempty"`
	Points  string   `json:"points,omitempty"`
}

type Answers map[string][]string

func AnswersOf(qs []Question) Answers {
	ans := make(Answers, len(qs))
	for _, q := range qs {
		ans[q.ID] = q.Answer
	}
	return ans
}

func (q Question) Chosen(o Option) bool {
	return slices.Contains(q.Answer, o.Value)
}

func (q Question) IsCorrect(o Option) bool {
	return slices.Contains(q.Correct, o.Value)
}

func (q Question) formValues(answer []string) []string {
	switch q.Kind {
	case KindRadio, KindCheckbox:
		return answer
	default:
		if len(answer) == 0 {
			return []string{""}
		}
		return answer[:1]
	}
}

var questionSpanRe = regexp.MustCompile(`^(qid\d+)question$`)

type segment struct {
	id    string
	nodes *goquery.Selection
}

type heading struct {
	index int
	id    string
}

func parseQuestions(p *Page) []Question {
	segs := segments(p.Doc.Find(".querypaper").First())
	qs := make([]Question, len(segs))
	for i, s := range segs {
		qs[i] = s.question(p)
	}
	return qs
}

func segments(paper *goquery.Selection) []segment {
	nodes := paper.Contents().Not(`script, .querybuttons, input[type="hidden"]`)

	var heads []heading
	for i, n := range nodes.EachIter() {
		if id := questionID(n); id != "" {
			heads = append(heads, heading{index: i, id: id})
		}
	}

	segs := make([]segment, len(heads))
	for i, h := range heads {
		end := nodes.Length()
		if i+1 < len(heads) {
			end = heads[i+1].index
		}
		segs[i] = segment{id: h.id, nodes: nodes.Slice(h.index, end)}
	}
	return segs
}

func questionID(n *goquery.Selection) string {
	ids := goquery.Map(findOrSelf(n, "span[id]"), attr("id"))
	for _, id := range ids {
		if m := submatch(questionSpanRe, id); m != "" {
			return m
		}
	}
	return ""
}

func (s segment) question(p *Page) Question {
	field := fmt.Sprintf(`[name=%q]`, s.id)
	q := parseControl(s.nodes, field)
	q.ID = s.id
	q.Text = s.text(field)
	q.Images = goquery.Map(findOrSelf(s.nodes, "img"), func(_ int, img *goquery.Selection) string {
		return p.Abs(img.AttrOr("src", ""))
	})
	q.Mark = text(s.nodes.Find(".answer"))
	q.Score = s.nodes.Find(".pointform input.point").AttrOr("value", "")
	return q
}

func (s segment) text(field string) string {
	prose := s.nodes.FilterFunction(func(_ int, n *goquery.Selection) bool {
		return findOrSelf(n, field+", .answer, .pointform, .iadroppedfiles").Length() == 0
	})
	lines := slices.DeleteFunc(goquery.Map(prose, func(_ int, n *goquery.Selection) string { return text(n) }),
		func(s string) bool { return s == "" })
	return strings.Join(lines, "\n")
}

func parseControl(nodes *goquery.Selection, field string) Question {
	if sel := findOrSelf(nodes, "select"+field); sel.Length() > 0 {
		return Question{
			Kind: KindSelect,
			Options: goquery.Map(sel.Find(`option:not([value=""])`), func(_ int, o *goquery.Selection) Option {
				return Option{Value: o.AttrOr("value", ""), Label: text(o)}
			}),
			Answer: goquery.Map(sel.Find(`option[selected]:not([value=""])`), attr("value")),
		}
	}
	if ta := findOrSelf(nodes, "textarea"+field); ta.Length() > 0 {
		return Question{Kind: KindTextarea, Answer: nonEmpty(ta.Text())}
	}
	inputs := findOrSelf(nodes, "input"+field)
	switch kind := Kind(inputs.AttrOr("type", "text")); kind {
	case KindRadio, KindCheckbox:
		return Question{
			Kind: kind,
			Options: goquery.Map(inputs, func(_ int, in *goquery.Selection) Option {
				return Option{Value: in.AttrOr("value", ""), Label: text(in.Parent().Find("label"))}
			}),
			Answer: goquery.Map(inputs.Filter("[checked]"), attr("value")),
		}
	default:
		return Question{Kind: KindText, Answer: nonEmpty(inputs.AttrOr("value", ""))}
	}
}

package ui

import "github.com/mst-mkt/rakutan/internal/manaba"

func Ask(q manaba.Question, current []string) ([]string, error) {
	switch q.Kind {
	case manaba.KindText:
		v, err := Input("回答", first(current))
		return nonEmpty(v), err
	case manaba.KindTextarea:
		v, err := Text("回答", first(current))
		return nonEmpty(v), err
	case manaba.KindCheckbox:
		return MultiSelect("回答 (複数選択可)", optionChoices(q), current)
	default:
		choices := append([]Choice[string]{{Label: "(未回答)", Value: ""}}, optionChoices(q)...)
		v, err := SelectFrom("回答", choices, first(current))
		return nonEmpty(v), err
	}
}

func optionChoices(q manaba.Question) []Choice[string] {
	choices := make([]Choice[string], len(q.Options))
	for i, o := range q.Options {
		choices[i] = Choice[string]{Label: o.Label, Value: o.Value}
	}
	return choices
}

func first(vs []string) string {
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}

func nonEmpty(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

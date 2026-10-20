package ui

import "testing"

func TestStyleKeepsOuterStyleAfterInnerReset(t *testing.T) {
	previous := colored
	colored = true
	t.Cleanup(func() { colored = previous })

	got := cyan("[" + red("7 時間") + "] 第4回")
	want := "\x1b[36m[\x1b[31m7 時間\x1b[0m\x1b[36m] 第4回\x1b[0m"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if w := displayWidth(got); w != 14 {
		t.Errorf("displayWidth = %d, want 14", w)
	}
}

package pantheon

import "testing"

func TestSeatNeedsInputQuestionVectors(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		wantOK   bool
		question string
		def      string
	}{
		{
			name:   "last line question",
			text:   "Did the first half.\nNEEDS INPUT: Which account should I use?\n\n",
			wantOK: true, question: "Which account should I use?",
		},
		{
			name:   "default line",
			text:   "Checked.\nNEEDS INPUT: CSV or Excel?\nDefault: CSV",
			wantOK: true, question: "CSV or Excel?", def: "CSV",
		},
		{
			name:   "not at line start",
			text:   "All done. NEEDS INPUT: not at line start",
			wantOK: false,
		},
		{
			name:   "empty question",
			text:   "NEEDS INPUT:",
			wantOK: false,
		},
		{
			name:   "multi line without needs input at end",
			text:   "NEEDS INPUT: first line\nsecond line",
			wantOK: false,
		},
		{
			name:   "fenced example at end does not trigger",
			text:   "正文。\n```\nNEEDS INPUT: 示例\n```",
			wantOK: false,
		},
		{
			name:   "quoted question at end does not trigger",
			text:   "正文。\n> NEEDS INPUT: 引用里的示例",
			wantOK: false,
		},
		{
			name:   "question before trailing blank lines",
			text:   "正文。\n\nNEEDS INPUT: 需要确认吗？\n\n",
			wantOK: true, question: "需要确认吗？",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := seatNeedsInputQuestion(tc.text)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (got %+v)", ok, tc.wantOK, got)
			}
			if !ok {
				return
			}
			if got.Question != tc.question || got.Default != tc.def {
				t.Fatalf("question = %+v, want question=%q default=%q", got, tc.question, tc.def)
			}
		})
	}
}

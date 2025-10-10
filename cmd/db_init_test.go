package cmd

import (
	"database/sql"
	"testing"

	"github.com/eslsoft/vocnet/internal/entity"
)

func Test_buildMeanings_alignment(t *testing.T) {
	w := wordRecord{
		Definition:  sql.NullString{String: "n. thing\nvt. do something\nvi. change", Valid: true},
		Translation: sql.NullString{String: "n. 东西\nvt. 做某事\nvi. 改变", Valid: true},
	}
	m, err := buildMeanings(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Fatal("expected meanings")
	}
	if len(m) != 6 {
		t.Fatalf("expected 6 meanings got %d", len(m))
	}
	// Definitions first
	if m[0].Pos != "n." || m[0].Text == "" || m[0].Language != entity.LanguageEnglish {
		t.Fatalf("bad first: %+v", m[0])
	}
	if m[1].Pos != "vt." || m[1].Text == "" || m[1].Language != entity.LanguageEnglish {
		t.Fatalf("bad second: %+v", m[1])
	}
	if m[2].Pos != "vi." || m[2].Text == "" || m[2].Language != entity.LanguageEnglish {
		t.Fatalf("bad third: %+v", m[2])
	}
	// Translations follow
	if m[3].Pos != "n." || m[3].Text == "" || m[3].Language != entity.LanguageChinese {
		t.Fatalf("bad fourth: %+v", m[3])
	}
	if m[4].Pos != "vt." || m[4].Text == "" || m[4].Language != entity.LanguageChinese {
		t.Fatalf("bad fifth: %+v", m[4])
	}
	if m[5].Pos != "vi." || m[5].Text == "" || m[5].Language != entity.LanguageChinese {
		t.Fatalf("bad sixth: %+v", m[5])
	}
}

func Test_extractLeadingPOS(t *testing.T) {
	cases := []struct{ in, pos, rest string }{
		{"vt. do sth", "vt.", "do sth"},
		{"v change", "v.", "change"},
		{"Adj. big", "adj.", "big"},           // case-insensitive
		{"noun something", "n.", "something"}, // 'n' followed by space
		{"adv. quickly", "adv.", "quickly"},
		{"no marker line", "", "no marker line"},
	}
	for _, c := range cases {
		p, r := extractLeadingPOS(c.in)
		if p != c.pos || r != c.rest {
			t.Fatalf("%q -> got (%q,%q) want (%q,%q)", c.in, p, r, c.pos, c.rest)
		}
	}
}

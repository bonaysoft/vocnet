package cmd

import (
	"database/sql"
	"encoding/json"
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
	if m == nil {
		t.Fatal("expected meanings")
	}
	var arr []jsonMeaning
	if err := json.Unmarshal(m.([]byte), &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 6 {
		t.Fatalf("expected 6 meanings got %d", len(arr))
	}
	// Definitions first
	if arr[0].Pos != "n" || arr[0].Text == "" || arr[0].Language != entity.LanguageEnglish.Code() {
		t.Fatalf("bad first: %+v", arr[0])
	}
	if arr[1].Pos != "vt" || arr[1].Text == "" || arr[1].Language != entity.LanguageEnglish.Code() {
		t.Fatalf("bad second: %+v", arr[1])
	}
	if arr[2].Pos != "vi" || arr[2].Text == "" || arr[2].Language != entity.LanguageEnglish.Code() {
		t.Fatalf("bad third: %+v", arr[2])
	}
	// Translations follow
	if arr[3].Pos != "n" || arr[3].Text == "" || arr[3].Language != entity.LanguageChinese.Code() {
		t.Fatalf("bad fourth: %+v", arr[3])
	}
	if arr[4].Pos != "vt" || arr[4].Text == "" || arr[4].Language != entity.LanguageChinese.Code() {
		t.Fatalf("bad fifth: %+v", arr[4])
	}
	if arr[5].Pos != "vi" || arr[5].Text == "" || arr[5].Language != entity.LanguageChinese.Code() {
		t.Fatalf("bad sixth: %+v", arr[5])
	}
}

func Test_extractLeadingPOS(t *testing.T) {
	cases := []struct{ in, pos, rest string }{
		{"vt. do sth", "vt", "do sth"},
		{"v change", "v", "change"},
		{"Adj. big", "adj", "big"},           // case-insensitive
		{"noun something", "n", "something"}, // 'n' followed by space
		{"adv. quickly", "adv", "quickly"},
		{"no marker line", "", "no marker line"},
	}
	for _, c := range cases {
		p, r := extractLeadingPOS(c.in)
		if p != c.pos || r != c.rest {
			t.Fatalf("%q -> got (%q,%q) want (%q,%q)", c.in, p, r, c.pos, c.rest)
		}
	}
}

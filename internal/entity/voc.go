package entity

import "time"

// Domain model now aligned with proto naming (Voc, VocMeaning, VocForm).
// This is the internal representation used by business logic.

type Voc struct {
	ID        int64        `json:"id"` // 自增ID, 基础CRUD用
	Text      string       `json:"text"`
	Language  string       `json:"language"`
	VocType   string       `json:"voc_type"`        // lemma, past, pp (past participle), ing (present participle), 3sg (third person singular), plural, comparative, superlative, variant, derived, other
	Lemma     *string      `json:"lemma,omitempty"` // nil if this row itself is the lemma
	Phonetic  string       `json:"phonetic,omitempty"`
	Meanings  []VocMeaning `json:"meanings,omitempty"` // only populated for lemma rows
	Tags      []string     `json:"tags,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	Forms     []VocFormRef `json:"forms,omitempty"` // if this is lemma: other forms; if not lemma: empty
}

type VocMeaning struct {
	POS         string `json:"pos"`         // n., v., adj.
	Definition  string `json:"definition"`  // English meaning/definition
	Translation string `json:"translation"` // Chinese translation / gloss
}

// Removed VocForm due to new flattened schema using voc_type + lemma linkage.

// VocFormRef is a lightweight reference to an inflected / variant form.
type VocFormRef struct {
	Text    string `json:"text"`
	VocType string `json:"voc_type"`
}

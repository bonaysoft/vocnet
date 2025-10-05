package entity

import (
	"time"
)

type Word struct {
	ID        int64          `json:"id"` // 自增ID, 基础CRUD用
	Text      string         `json:"text"`
	Language  Language       `json:"language"`
	WordType  string         `json:"word_type"`       // lemma, past, pp (past participle), ing (present participle), 3sg (third person singular), plural, comparative, superlative, variant, derived, other
	Lemma     *string        `json:"lemma,omitempty"` // nil if this row itself is the lemma
	Phonetics []WordPhonetic `json:"phonetics,omitempty"`

	Definitions []WordDefinition `json:"definitions,omitempty"` // only populated for lemma rows
	Tags        []string         `json:"tags,omitempty"`
	Phrases     []Phrase         `json:"phrases,omitempty"`
	Sentences   []Sentence       `json:"sentences,omitempty"`
	Relations   []WordRelation   `json:"relations,omitempty"`
	Forms       []WordFormRef    `json:"forms,omitempty"` // if this is lemma: other forms; if not lemma: empty

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WordPhonetic struct {
	IPA     string `json:"ipa"`
	Dialect string `json:"dialect,omitempty"`
}

type WordDefinition struct {
	Pos      string   `json:"pos"`
	Text     string   `json:"text"`
	Language Language `json:"language"`
}

// Sentence captures a short contextual example recorded by the user.
type Sentence struct {
	Text      string `json:"text"`
	Source    int32  `json:"source"`
	SourceRef string `json:"source_ref,omitempty"`
}

type WordFormRef struct {
	Text     string `json:"text"`
	WordType string `json:"word_type"`
}

// WordRelation models a connection to another dictionary entry.
type WordRelation struct {
	Word         string `json:"word"`
	RelationType int32  `json:"relation_type"`
}

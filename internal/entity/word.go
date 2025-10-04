package entity

import (
	"strings"
	"time"
)

// Language represents supported language codes using ISO-style abbreviations.
type Language string

const (
	LanguageUnspecified Language = ""
	LanguageEnglish     Language = "en"
	LanguageChinese     Language = "zh"
	LanguageSpanish     Language = "es"
	LanguageFrench      Language = "fr"
	LanguageGerman      Language = "de"
	LanguageJapanese    Language = "ja"
	LanguageKorean      Language = "ko"
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
	Phrases     []string         `json:"phrases,omitempty"`
	Sentences   []Sentence       `json:"sentences,omitempty"`
	Relations   []WordRelation   `json:"relations,omitempty"`
	Forms       []WordFormRef    `json:"forms,omitempty"` // if this is lemma: other forms; if not lemma: empty

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WordDefinition struct {
	Pos      string   `json:"pos"`
	Text     string   `json:"text"`
	Language Language `json:"language"`
}

type WordPhonetic struct {
	IPA     string `json:"ipa"`
	Dialect string `json:"dialect,omitempty"`
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

// WordFilter defines filtering options when listing vocabulary entries.
type WordFilter struct {
	Pagination

	Language Language
	Keyword  string
	WordType string
	Words    []string
}

// NormalizeLanguage ensures the language falls back to a supported value (defaults to English).
func NormalizeLanguage(lang Language) Language {
	switch lang {
	case LanguageEnglish, LanguageChinese, LanguageSpanish, LanguageFrench, LanguageGerman, LanguageJapanese, LanguageKorean:
		return lang
	default:
		return LanguageEnglish
	}
}

// ParseLanguage converts an arbitrary string into a supported Language value.
func ParseLanguage(code string) Language {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "en":
		return LanguageEnglish
	case "zh":
		return LanguageChinese
	case "es":
		return LanguageSpanish
	case "fr":
		return LanguageFrench
	case "de":
		return LanguageGerman
	case "ja":
		return LanguageJapanese
	case "ko":
		return LanguageKorean
	case "":
		return LanguageUnspecified
	default:
		return LanguageUnspecified
	}
}

// Code returns the lowercase language code (without defaulting).
func (l Language) Code() string {
	return strings.TrimSpace(string(l))
}

// CodeOrDefault returns the language code, falling back to English when unspecified.
func (l Language) CodeOrDefault() string {
	if l.Code() == "" {
		return string(LanguageEnglish)
	}
	return l.Code()
}

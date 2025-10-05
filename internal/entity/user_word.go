package entity

import "time"

// UserWord represents a user's personalised vocabulary entry.
type UserWord struct {
	ID         int64
	UserID     int64
	Word       string
	Language   Language
	Mastery    MasteryBreakdown
	Review     ReviewTiming
	QueryCount int64
	Notes      string
	Sentences  []Sentence
	Relations  []UserWordRelation
	CreatedBy  string
	CreatedAt  time.Time
	UpdatedAt  time.Time

	WordContent *Word // Optional linked dictionary word entry
}

// MasteryBreakdown captures skill-specific mastery scores for a user word.
type MasteryBreakdown struct {
	Listen    int32
	Read      int32
	Spell     int32
	Pronounce int32
	Use       int32
	Overall   int32
}

// ReviewTiming represents spaced repetition metadata for a user word.
type ReviewTiming struct {
	LastReviewAt *time.Time
	NextReviewAt *time.Time
	IntervalDays int32
	FailCount    int32
}

// UserWordRelation links a user word to another concept in their vocabulary graph.
type UserWordRelation struct {
	Word         string    `json:"word"`
	RelationType int32     `json:"relation_type"`
	Note         string    `json:"note,omitempty"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Normalize ensures defaults & constraints before persistence.
func (uw *UserWord) Normalize(now time.Time) {
	if uw.CreatedAt.IsZero() {
		uw.CreatedAt = now
	}
	uw.UpdatedAt = now
	if uw.Language == "" {
		uw.Language = "en"
	}
	if uw.Sentences == nil {
		uw.Sentences = []Sentence{}
	}
	if uw.Relations == nil {
		uw.Relations = []UserWordRelation{}
	}
}

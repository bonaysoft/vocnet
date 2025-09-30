package entity

import "time"

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

// Sentence captures a short contextual example recorded by the user.
type Sentence struct {
	Text   string
	Source int32
}

// WordRelation links a user word to another concept in their vocabulary graph.
type WordRelation struct {
	Word         string
	RelationType int32
	Note         string
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserWord represents a user's personalised vocabulary entry.
type UserWord struct {
	ID         int64
	UserID     int64
	Word       string
	Language   string
	Mastery    MasteryBreakdown
	Review     ReviewTiming
	QueryCount int64
	Notes      string
	Sentences  []Sentence
	Relations  []WordRelation
	CreatedBy  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// UserWordFilter allows searching for user words in repository implementations.
type UserWordFilter struct {
	UserID  int64
	Keyword string
	Limit   int32
	Offset  int32
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
		uw.Relations = []WordRelation{}
	}
}

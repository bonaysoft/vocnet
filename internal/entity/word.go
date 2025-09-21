package entity

import "time"

// Word represents a global vocabulary entry in the domain layer.
// Keep this decoupled from protobuf/generated DB structs for clean architecture.
type Word struct {
	ID          int64
	Lemma       string
	Language    string
	Phonetic    string
	POS         string
	Definition  string
	Translation string
	Exchange    string
	Tags        []string
	CreatedAt   time.Time
}

package entity

import "time"

type Phrase struct {
	ID          int64            `json:"id"`
	Text        string           `json:"text"`
	Language    Language         `json:"language"`
	Definitions []WordDefinition `json:"definitions"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

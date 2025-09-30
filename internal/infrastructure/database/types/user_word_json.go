package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// UserSentence represents a stored sentence tied to a user word.
type UserSentence struct {
	Text   string `json:"text"`
	Source int32  `json:"source"`
}

// UserWordRelation represents a relation entry stored alongside a user word.
type UserWordRelation struct {
	Word         string    `json:"word"`
	RelationType int32     `json:"relation_type"`
	Note         string    `json:"note"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserSentences []UserSentence
type UserWordRelations []UserWordRelation

// Scan implements sql.Scanner for UserSentences.
func (s *UserSentences) Scan(src any) error {
	if src == nil {
		*s = nil
		return nil
	}
	switch data := src.(type) {
	case []byte:
		if len(data) == 0 {
			*s = nil
			return nil
		}
		return json.Unmarshal(data, s)
	case string:
		if data == "" {
			*s = nil
			return nil
		}
		return json.Unmarshal([]byte(data), s)
	default:
		return fmt.Errorf("UserSentences: unsupported src type %T", src)
	}
}

// Value implements driver.Valuer for UserSentences.
func (s UserSentences) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Scan implements sql.Scanner for UserWordRelations.
func (r *UserWordRelations) Scan(src any) error {
	if src == nil {
		*r = nil
		return nil
	}
	switch data := src.(type) {
	case []byte:
		if len(data) == 0 {
			*r = nil
			return nil
		}
		return json.Unmarshal(data, r)
	case string:
		if data == "" {
			*r = nil
			return nil
		}
		return json.Unmarshal([]byte(data), r)
	default:
		return fmt.Errorf("UserWordRelations: unsupported src type %T", src)
	}
}

// Value implements driver.Valuer for UserWordRelations.
func (r UserWordRelations) Value() (driver.Value, error) {
	if r == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return b, nil
}

package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/eslsoft/vocnet/internal/entity"
)

type WordMeanings []entity.WordDefinition

type WordPhonetics []entity.WordPhonetic

// Scan implements sql.Scanner
func (v *WordMeanings) Scan(src any) error {
	if src == nil {
		*v = nil
		return nil
	}
	switch data := src.(type) {
	case []byte:
		if len(data) == 0 {
			*v = nil
			return nil
		}
		return json.Unmarshal(data, v)
	case string:
		if data == "" {
			*v = nil
			return nil
		}
		return json.Unmarshal([]byte(data), v)
	default:
		return fmt.Errorf("VocMeanings: unsupported src type %T", src)
	}
}

// Value implements driver.Valuer
func (v WordMeanings) Value() (driver.Value, error) {
	if v == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Scan implements sql.Scanner
func (p *WordPhonetics) Scan(src any) error {
	if src == nil {
		*p = nil
		return nil
	}
	switch data := src.(type) {
	case []byte:
		if len(data) == 0 {
			*p = nil
			return nil
		}
		return json.Unmarshal(data, p)
	case string:
		if data == "" {
			*p = nil
			return nil
		}
		return json.Unmarshal([]byte(data), p)
	default:
		return fmt.Errorf("WordPhonetics: unsupported src type %T", src)
	}
}

// Value implements driver.Valuer
func (p WordPhonetics) Value() (driver.Value, error) {
	if p == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type WordForms []entity.WordFormRef

// Scan implements sql.Scanner
func (f *WordForms) Scan(src any) error {
	if src == nil {
		*f = nil
		return nil
	}
	switch data := src.(type) {
	case []byte:
		if len(data) == 0 {
			*f = nil
			return nil
		}
		return json.Unmarshal(data, f)
	case string:
		if data == "" {
			*f = nil
			return nil
		}
		return json.Unmarshal([]byte(data), f)
	default:
		return fmt.Errorf("VocForms: unsupported src type %T", src)
	}
}

// Value implements driver.Valuer
func (f WordForms) Value() (driver.Value, error) {
	if f == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

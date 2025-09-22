package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// VocMeaning mirrors JSON structure stored in vocs.meanings
type VocMeaning struct {
	POS         string `json:"pos"`
	Definition  string `json:"definition"`
	Translation string `json:"translation"`
}

// VocForm mirrors JSON structure stored in vocs.forms
type VocForm struct {
	Word string `json:"word"`
	Type string `json:"type"`
}

type VocMeanings []VocMeaning
type VocForms []VocForm

// Scan implements sql.Scanner
func (v *VocMeanings) Scan(src any) error {
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
func (v VocMeanings) Value() (driver.Value, error) {
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
func (f *VocForms) Scan(src any) error {
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
func (f VocForms) Value() (driver.Value, error) {
	if f == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

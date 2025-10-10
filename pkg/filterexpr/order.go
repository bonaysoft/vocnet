package filterexpr

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type orderParams struct {
	PrimaryKey    string
	PrimaryDesc   bool
	SecondaryKey  string
	SecondaryDesc bool
}

func parseOrderBy(raw string, schema OrderSchema) (orderParams, error) { //nolint:gocognit,gocyclo // parsing DSL entails validation branches for readability
	if schema.Fields == nil {
		schema.Fields = map[string]OrderField{}
	}

	if schema.DefaultPrimary == "" {
		return orderParams{}, errors.New("order schema default primary key required")
	}
	if schema.FallbackKey == "" {
		return orderParams{}, errors.New("order schema fallback key required")
	}

	if _, ok := schema.Fields[schema.DefaultPrimary]; !ok {
		return orderParams{}, fmt.Errorf("order key %q missing from schema fields", schema.DefaultPrimary)
	}
	if _, ok := schema.Fields[schema.FallbackKey]; !ok {
		return orderParams{}, fmt.Errorf("fallback order key %q missing from schema fields", schema.FallbackKey)
	}

	ord := orderParams{
		PrimaryKey:    schema.DefaultPrimary,
		PrimaryDesc:   schema.DefaultPrimaryDesc,
		SecondaryKey:  schema.FallbackKey,
		SecondaryDesc: schema.FallbackDesc,
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ord, nil
	}

	segments := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(segments))
	idx := 0
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		parts := strings.Fields(seg)
		if len(parts) == 0 {
			continue
		}
		key := parts[0]
		if _, ok := schema.Fields[key]; !ok {
			return orderParams{}, fmt.Errorf("field %q cannot be used for ordering", key)
		}

		var desc bool
		switch len(parts) {
		case 1:
			desc = false
		case 2:
			dir := strings.ToLower(parts[1])
			switch dir {
			case "asc":
				desc = false
			case "desc":
				desc = true
			default:
				return orderParams{}, fmt.Errorf("invalid direction %q for field %q", parts[1], key)
			}
		default:
			return orderParams{}, fmt.Errorf("invalid order segment %q", seg)
		}

		if _, dup := seen[key]; dup {
			return orderParams{}, fmt.Errorf("duplicate order key %q", key)
		}
		seen[key] = struct{}{}

		switch idx {
		case 0:
			ord.PrimaryKey = key
			ord.PrimaryDesc = desc
		case 1:
			ord.SecondaryKey = key
			ord.SecondaryDesc = desc
		default:
			return orderParams{}, errors.New("order_by supports at most two keys")
		}
		idx++
	}

	if ord.SecondaryKey == "" {
		ord.SecondaryKey = schema.FallbackKey
		ord.SecondaryDesc = schema.FallbackDesc
	}

	if ord.SecondaryKey == ord.PrimaryKey {
		// ensure deterministic ordering by falling back to default primary when fallback duplicates primary
		for key := range schema.Fields {
			if key != ord.PrimaryKey {
				ord.SecondaryKey = key
				ord.SecondaryDesc = false
				break
			}
		}
		if ord.SecondaryKey == ord.PrimaryKey {
			return orderParams{}, errors.New("order schema requires at least two distinct keys for stable ordering")
		}
	}

	return ord, nil
}

func setOrderParams(binding any, ord orderParams) error {
	rv := reflect.ValueOf(binding)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("binding must be a non-nil pointer")
	}

	target := rv.Elem()
	if target.Kind() != reflect.Struct {
		return errors.New("binding must point to a struct")
	}

	if err := setStringField(target, "PrimaryKey", ord.PrimaryKey); err != nil {
		return err
	}
	if err := setBoolField(target, "PrimaryDesc", ord.PrimaryDesc); err != nil {
		return err
	}
	if err := setStringField(target, "SecondaryKey", ord.SecondaryKey); err != nil {
		return err
	}
	if err := setBoolField(target, "SecondaryDesc", ord.SecondaryDesc); err != nil {
		return err
	}

	return nil
}

func setStringField(target reflect.Value, name string, value string) error {
	return setAssignableField(target, name, reflect.ValueOf(value))
}

func setBoolField(target reflect.Value, name string, value bool) error {
	return setAssignableField(target, name, reflect.ValueOf(value))
}

func setAssignableField(target reflect.Value, name string, value reflect.Value) error {
	field := target.FieldByName(name)
	if !field.IsValid() {
		return fmt.Errorf("params struct %s has no field named %q", target.Type(), name)
	}
	if !field.CanSet() {
		return fmt.Errorf("cannot set field %q on params struct", name)
	}

	switch field.Kind() {
	case reflect.Interface:
		field.Set(value)
		return nil
	case reflect.Ptr:
		elemType := field.Type().Elem()
		if !value.Type().ConvertibleTo(elemType) {
			return fmt.Errorf("field %q must be %s-compatible, got %s", name, elemType, value.Type())
		}
		if field.IsNil() {
			field.Set(reflect.New(elemType))
		}
		field.Elem().Set(value.Convert(elemType))
		return nil
	default:
		if !value.Type().ConvertibleTo(field.Type()) {
			return fmt.Errorf("field %q must be %s-compatible, got %s", name, field.Type(), value.Type())
		}
		field.Set(value.Convert(field.Type()))
		return nil
	}
}

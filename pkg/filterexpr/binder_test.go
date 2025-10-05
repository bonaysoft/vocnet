package filterexpr

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type ListItemsParams struct {
	State        *string
	PriceMin     *float64
	PriceMax     *float64
	NamePrefix   *string
	CreatedAfter *time.Time
	Limit        int32
	Offset       int32
}

var ItemsSchema = Schema{
	Fields: map[string]FieldRule{
		"state": {
			Kind: KindString,
			Ops:  map[Op]string{OpEQ: "State"},
		},
		"price": {
			Kind: KindNumber,
			Ops: map[Op]string{
				OpGTE: "PriceMin",
				OpLTE: "PriceMax",
			},
		},
		"name": {
			Kind: KindString,
			Ops:  map[Op]string{OpSW: "NamePrefix"},
		},
		"create_time": {
			Kind: KindTimestamp,
			Ops:  map[Op]string{OpGTE: "CreatedAfter"},
		},
	},
}

func TestBindCELTo_ListItems(t *testing.T) {
	var params ListItemsParams
	timestamp := "2025-01-01T00:00:00Z"
	filter := fmt.Sprintf("state == 'ACTIVE' && price <= 1000 && name.startsWith('A') && create_time >= timestamp('%s')", timestamp)

	if err := BindCELTo(filter, &params, ItemsSchema); err != nil {
		t.Fatalf("BindCELTo returned error: %v", err)
	}

	if params.State == nil || *params.State != "ACTIVE" {
		t.Fatalf("expected State to be 'ACTIVE', got %v", params.State)
	}
	if params.PriceMax == nil || *params.PriceMax != 1000 {
		t.Fatalf("expected PriceMax to be 1000, got %v", params.PriceMax)
	}
	if params.PriceMin != nil {
		t.Fatalf("expected PriceMin to be nil, got %v", params.PriceMin)
	}
	if params.NamePrefix == nil || *params.NamePrefix != "A" {
		t.Fatalf("expected NamePrefix to be 'A', got %v", params.NamePrefix)
	}
	if params.CreatedAfter == nil {
		t.Fatalf("expected CreatedAfter to be set")
	}

	wantTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !params.CreatedAfter.Equal(wantTime) {
		t.Fatalf("expected CreatedAfter %v, got %v", wantTime, params.CreatedAfter)
	}
}

func TestBindCELTo_NumberBounds(t *testing.T) {
	var params ListItemsParams
	filter := "price >= 10 && price <= 20"

	if err := BindCELTo(filter, &params, ItemsSchema); err != nil {
		t.Fatalf("BindCELTo returned error: %v", err)
	}

	if params.PriceMin == nil || *params.PriceMin != 10 {
		t.Fatalf("expected PriceMin 10, got %v", params.PriceMin)
	}
	if params.PriceMax == nil || *params.PriceMax != 20 {
		t.Fatalf("expected PriceMax 20, got %v", params.PriceMax)
	}
}

func TestBindCELTo_ReceiverStartsWith(t *testing.T) {
	var params ListItemsParams
	filter := "name.startsWith('Pre')"

	if err := BindCELTo(filter, &params, ItemsSchema); err != nil {
		t.Fatalf("BindCELTo returned error: %v", err)
	}

	if params.NamePrefix == nil || *params.NamePrefix != "Pre" {
		t.Fatalf("expected NamePrefix 'Pre', got %v", params.NamePrefix)
	}
}

func TestBindCELTo_CustomSetter(t *testing.T) {
	type WithPG struct {
		State pgtype.Text
	}

	schema := Schema{
		Fields: map[string]FieldRule{
			"state": {
				Kind: KindString,
				Ops:  map[Op]string{OpEQ: "State"},
				Setter: func(field reflect.Value, v any) error {
					text, ok := v.(string)
					if !ok {
						return fmt.Errorf("expected string, got %T", v)
					}
					ft := field.Interface().(pgtype.Text)
					ft.String = text
					ft.Valid = true
					field.Set(reflect.ValueOf(ft))
					return nil
				},
			},
		},
	}

	var params WithPG
	if err := BindCELTo("state == 'ACTIVE'", &params, schema); err != nil {
		t.Fatalf("BindCELTo returned error: %v", err)
	}

	if !params.State.Valid || params.State.String != "ACTIVE" {
		t.Fatalf("expected state ACTIVE, got %+v", params.State)
	}
}

func TestBindCELTo_InOperator(t *testing.T) {
	type Params struct {
		Names []string
	}

	schema := Schema{
		Fields: map[string]FieldRule{
			"name": {
				Kind: KindString,
				Ops:  map[Op]string{OpIN: "Names"},
			},
		},
	}

	var params Params
	if err := BindCELTo("name in ['Alice', 'Bob']", &params, schema); err != nil {
		t.Fatalf("BindCELTo returned error: %v", err)
	}

	want := []string{"Alice", "Bob"}
	if !reflect.DeepEqual(params.Names, want) {
		t.Fatalf("expected Names %v, got %v", want, params.Names)
	}
}

func TestBindCELTo_Errors(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		want   string
	}{
		{"unsupported field", "unknown == 'x'", "not allowed"},
		{"unsupported operator", "state <= 'A'", "operator"},
		{"bad literal type", "state == 1", "expected string"},
		{"bad logical op", "state == 'A' || price <= 10", "only AND"},
		{"non literal", "price <= foo", "right-hand side"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var params ListItemsParams
			err := BindCELTo(tc.filter, &params, ItemsSchema)
			if err == nil {
				t.Fatalf("expected error for %q", tc.filter)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("expected error to contain %q, got %v", tc.want, err)
			}
		})
	}
}

func TestBindCELTo_ListWrongType(t *testing.T) {
	schema := Schema{
		Fields: map[string]FieldRule{
			"state": {
				Kind: KindString,
				Ops:  map[Op]string{OpIN: "States"},
			},
		},
	}

	type params struct {
		States []string
	}

	var p params
	err := BindCELTo("state in [1]", &p, schema)
	if err == nil || !strings.Contains(err.Error(), "list literal elements must be strings") {
		t.Fatalf("expected list literal error, got %v", err)
	}
}

func TestBindCELTo_InvalidParams(t *testing.T) {
	var params *ListItemsParams
	if err := BindCELTo("state == 'ACTIVE'", params, ItemsSchema); err == nil {
		t.Fatalf("expected error when params is nil pointer")
	}
}

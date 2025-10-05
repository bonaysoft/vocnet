package filterexpr

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type listMsg struct {
	filter  string
	orderBy string
}

func (m listMsg) GetFilter() string  { return m.filter }
func (m listMsg) GetOrderBy() string { return m.orderBy }

type listParams struct {
	State         *string
	PriceMin      *float64
	PriceMax      *float64
	NamePrefix    *string
	CreatedAfter  *time.Time
	Names         []string
	PrimaryKey    string
	PrimaryDesc   bool
	SecondaryKey  string
	SecondaryDesc bool
}

var testSchema = ResourceSchema{
	Filter: map[string]FilterField{
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
			Ops: map[Op]string{
				OpSW: "NamePrefix",
				OpIN: "Names",
			},
		},
		"create_time": {
			Kind: KindTimestamp,
			Ops:  map[Op]string{OpGTE: "CreatedAfter"},
		},
	},
	Order: OrderSchema{
		DefaultPrimary:     "create_time",
		DefaultPrimaryDesc: true,
		FallbackKey:        "id",
		FallbackDesc:       false,
		Fields: map[string]OrderField{
			"create_time": {Expr: "create_time", Nulls: "last"},
			"updated_at":  {Expr: "updated_at", Nulls: "last"},
			"text":        {Expr: "text", Nulls: "last"},
			"id":          {Expr: "id", Nulls: "last"},
		},
	},
}

func TestBind_FilterAndOrder(t *testing.T) {
	timestamp := "2025-01-01T00:00:00Z"
	msg := listMsg{
		filter:  fmt.Sprintf("state == 'ACTIVE' && price <= 1000 && name.startsWith('A') && create_time >= timestamp('%s')", timestamp),
		orderBy: "text asc, updated_at desc",
	}

	var params listParams
	if err := Bind(msg, &params, testSchema); err != nil {
		t.Fatalf("Bind returned error: %v", err)
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

	if params.PrimaryKey != "text" || params.PrimaryDesc {
		t.Fatalf("expected primary order text asc, got key=%q desc=%v", params.PrimaryKey, params.PrimaryDesc)
	}
	if params.SecondaryKey != "updated_at" || !params.SecondaryDesc {
		t.Fatalf("expected secondary order updated_at desc, got key=%q desc=%v", params.SecondaryKey, params.SecondaryDesc)
	}
}

func TestBind_CustomSetter(t *testing.T) {
	type withPG struct {
		State         pgtype.Text
		PrimaryKey    string
		PrimaryDesc   bool
		SecondaryKey  string
		SecondaryDesc bool
	}

	schema := ResourceSchema{
		Filter: map[string]FilterField{
			"state": {
				Kind: KindString,
				Ops:  map[Op]string{OpEQ: "State"},
				Setter: func(field reflect.Value, v any) error {
					text, ok := v.(string)
					if !ok {
						return fmt.Errorf("expected string, got %T", v)
					}
					t := field.Interface().(pgtype.Text)
					t.String = text
					t.Valid = true
					field.Set(reflect.ValueOf(t))
					return nil
				},
			},
		},
		Order: testSchema.Order,
	}

	msg := listMsg{filter: "state == 'ACTIVE'"}
	var params withPG
	if err := Bind(msg, &params, schema); err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if !params.State.Valid || params.State.String != "ACTIVE" {
		t.Fatalf("expected state ACTIVE, got %+v", params.State)
	}
}

func TestBind_InOperator(t *testing.T) {
	msg := listMsg{filter: "name in ['Alice', 'Bob']"}

	var params listParams
	if err := Bind(msg, &params, testSchema); err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	want := []string{"Alice", "Bob"}
	if !reflect.DeepEqual(params.Names, want) {
		t.Fatalf("expected Names %v, got %v", want, params.Names)
	}
}

func TestBind_OrderDefaults(t *testing.T) {
	var params listParams
	if err := Bind(listMsg{}, &params, testSchema); err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if params.PrimaryKey != "create_time" || !params.PrimaryDesc {
		t.Fatalf("expected default primary create_time desc, got key=%q desc=%v", params.PrimaryKey, params.PrimaryDesc)
	}
	if params.SecondaryKey != "id" || params.SecondaryDesc {
		t.Fatalf("expected fallback id asc, got key=%q desc=%v", params.SecondaryKey, params.SecondaryDesc)
	}
}

func TestBind_OrderErrors(t *testing.T) {
	tests := []struct {
		name    string
		orderBy string
		want    string
	}{
		{"unknown field", "rating desc", "cannot be used"},
		{"bad direction", "text downward", "invalid direction"},
		{"too many keys", "text asc, updated_at desc, create_time asc", "at most two"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var params listParams
			err := Bind(listMsg{orderBy: tc.orderBy}, &params, testSchema)
			if err == nil {
				t.Fatalf("expected error for order_by %q", tc.orderBy)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("expected error to contain %q, got %v", tc.want, err)
			}
		})
	}
}

func TestBind_FilterErrors(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		want   string
	}{
		{"unsupported field", "unknown == 'x'", "not allowed"},
		{"unsupported operator", "state <= 'A'", "operator"},
		{"bad literal type", "state == 1", "expected string"},
		{"bad logical op", "state == 'A' || price <= 10", "only AND"},
		{"non literal", "price <= foo", "right-hand"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var params listParams
			err := Bind(listMsg{filter: tc.filter}, &params, testSchema)
			if err == nil {
				t.Fatalf("expected error for %q", tc.filter)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("expected error to contain %q, got %v", tc.want, err)
			}
		})
	}
}

func TestBind_InvalidBinding(t *testing.T) {
	var params *listParams
	if err := Bind(listMsg{filter: "state == 'ACTIVE'"}, params, testSchema); err == nil {
		t.Fatalf("expected error when binding is nil pointer")
	}
}

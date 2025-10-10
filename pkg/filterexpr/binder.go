package filterexpr

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// Msg wraps request DTOs that expose filter and order_by raw inputs.
type Msg interface {
	GetFilter() string
	GetOrderBy() string
}

// ValueKind describes the kind of literal value a field accepts.
type ValueKind string

const (
	KindString    ValueKind = "string"
	KindNumber    ValueKind = "number"
	KindTimestamp ValueKind = "timestamp"
)

// Op represents a supported comparison operation.
type Op string

const (
	OpEQ  Op = "=="
	OpGTE Op = ">="
	OpLTE Op = "<="
	OpSW  Op = "startsWith"
	OpIN  Op = "in"
)

// SetterFunc allows custom assignment of literal values to struct fields.
type SetterFunc func(field reflect.Value, value any) error

// FilterField describes how a filter field maps to a params struct field and which operations are allowed.
type FilterField struct {
	Expr   string
	Kind   ValueKind
	Ops    map[Op]string
	Setter SetterFunc
}

// OrderField maps an order key to a SQL expression.
type OrderField struct {
	Expr  string
	Nulls string
}

// OrderSchema describes ordering defaults and whitelisted keys.
type OrderSchema struct {
	DefaultPrimary     string
	DefaultPrimaryDesc bool
	FallbackKey        string
	FallbackDesc       bool
	Fields             map[string]OrderField
}

// ResourceSchema aggregates filtering and ordering rules for a resource.
type ResourceSchema struct {
	Filter map[string]FilterField
	Order  OrderSchema
}

var timeType = reflect.TypeOf(time.Time{})

// Bind parses the request filter & order_by and populates the query params struct accordingly.
func Bind[M Msg, P any](msg M, binding *P, schema ResourceSchema) error {
	if binding == nil {
		return errors.New("binding must not be nil")
	}

	if err := bindFilterTo(binding, msg.GetFilter(), schema.Filter); err != nil {
		return fmt.Errorf("filter: %w", err)
	}

	order, err := parseOrderBy(msg.GetOrderBy(), schema.Order)
	if err != nil {
		return fmt.Errorf("order_by: %w", err)
	}

	if err := setOrderParams(binding, order); err != nil {
		return err
	}

	return nil
}

func bindFilterTo(binding any, filter string, fields map[string]FilterField) error {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return nil
	}

	if len(fields) == 0 {
		return errors.New("filter schema has no fields defined")
	}

	env, err := buildEnv(fields)
	if err != nil {
		return err
	}

	ast, issues := env.Parse(filter)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("invalid filter: %w", issues.Err())
	}

	parsed, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return fmt.Errorf("failed to convert AST: %w", err)
	}
	conjuncts, err := extractConjuncts(parsed.GetExpr())
	if err != nil {
		return err
	}

	paramsVal := reflect.ValueOf(binding)
	if paramsVal.Kind() != reflect.Ptr || paramsVal.IsNil() {
		return errors.New("binding must be a non-nil pointer")
	}

	dest := paramsVal.Elem()
	if dest.Kind() != reflect.Struct {
		return errors.New("binding must point to a struct")
	}

	for _, expr := range conjuncts {
		pred, err := parseAtomicPredicate(expr)
		if err != nil {
			return err
		}

		rule, ok := fields[pred.Field]
		if !ok {
			return fmt.Errorf("field %q is not allowed", pred.Field)
		}

		targetName, ok := rule.Ops[pred.Op]
		if !ok {
			return fmt.Errorf("operator %q is not allowed for field %q", string(pred.Op), pred.Field)
		}

		if err := validateLiteral(rule.Kind, pred.Op, pred.Value); err != nil {
			return fmt.Errorf("field %q: %w", pred.Field, err)
		}

		field := dest.FieldByName(targetName)
		if !field.IsValid() {
			return fmt.Errorf("params struct %s has no field named %q", dest.Type(), targetName)
		}
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %q on params struct", targetName)
		}

		if rule.Setter != nil {
			if err := callSetter(rule.Setter, field, pred.Value); err != nil {
				return fmt.Errorf("setter for field %q failed: %w", targetName, err)
			}
			continue
		}

		if err := assignValue(field, pred.Value); err != nil {
			return fmt.Errorf("failed to assign field %q: %w", targetName, err)
		}
	}

	return nil
}

type atomicPredicate struct {
	Field string
	Op    Op
	Value any
}

func buildEnv(fields map[string]FilterField) (*cel.Env, error) {
	opts := make([]cel.EnvOption, 0, len(fields))
	for name, rule := range fields {
		celType, err := celTypeForKind(rule.Kind)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", name, err)
		}
		opts = append(opts, cel.Variable(name, celType))
	}
	opts = append(opts, cel.CrossTypeNumericComparisons(true))

	// NOTE: cel-go v0.26.1 does not export an EnvOption for variadic logical operators.
	// We accept the default binary AST shape and flatten nested AND chains in extractConjuncts.
	return cel.NewEnv(opts...)
}

func celTypeForKind(kind ValueKind) (*cel.Type, error) {
	switch kind {
	case KindString:
		return cel.StringType, nil
	case KindNumber:
		return cel.DoubleType, nil
	case KindTimestamp:
		return cel.TimestampType, nil
	default:
		return nil, fmt.Errorf("unsupported field kind %s", kind)
	}
}

func extractConjuncts(expr *exprpb.Expr) ([]*exprpb.Expr, error) {
	if expr == nil {
		return nil, errors.New("empty expression")
	}

	call := expr.GetCallExpr()
	if call == nil {
		return []*exprpb.Expr{expr}, nil
	}

	switch call.Function {
	case "_&&_":
		if len(call.Args) < 2 || call.Target != nil {
			return nil, errors.New("logical AND must have at least two operands")
		}
		var result []*exprpb.Expr
		for _, arg := range call.Args {
			conjuncts, err := extractConjuncts(arg)
			if err != nil {
				return nil, err
			}
			result = append(result, conjuncts...)
		}
		return result, nil
	case "_||_", "_?_:_", "!":
		return nil, fmt.Errorf("logical operator %q is not supported; only AND is allowed", call.Function)
	default:
		return []*exprpb.Expr{expr}, nil
	}
}

func parseAtomicPredicate(expr *exprpb.Expr) (atomicPredicate, error) {
	call := expr.GetCallExpr()
	if call == nil {
		return atomicPredicate{}, errors.New("unsupported expression; expected comparison or function call")
	}

	switch call.Function {
	case "_==_":
		return parseBinaryPredicate(call, OpEQ)
	case "_>=_":
		return parseBinaryPredicate(call, OpGTE)
	case "_<=_":
		return parseBinaryPredicate(call, OpLTE)
	case "_in_", "@in":
		return parseInPredicate(call)
	case "startsWith":
		return parseStartsWith(call)
	default:
		return atomicPredicate{}, fmt.Errorf("function %q is not supported", call.Function)
	}
}

func parseBinaryPredicate(call *exprpb.Expr_Call, op Op) (atomicPredicate, error) {
	if call.Target != nil || len(call.Args) != 2 {
		return atomicPredicate{}, fmt.Errorf("operator %q expects two operands", string(op))
	}

	fieldName, err := parseFieldIdent(call.Args[0])
	if err != nil {
		return atomicPredicate{}, err
	}

	value, err := parseLiteral(call.Args[1])
	if err != nil {
		return atomicPredicate{}, err
	}

	return atomicPredicate{Field: fieldName, Op: op, Value: value}, nil
}

func parseInPredicate(call *exprpb.Expr_Call) (atomicPredicate, error) {
	var fieldExpr *exprpb.Expr
	var listExpr *exprpb.Expr

	if call.Target != nil {
		if len(call.Args) != 1 {
			return atomicPredicate{}, errors.New("in operator with receiver must have exactly one argument")
		}
		listExpr = call.Target
		fieldExpr = call.Args[0]
	} else {
		if len(call.Args) != 2 {
			return atomicPredicate{}, errors.New("in operator expects two operands")
		}
		fieldExpr = call.Args[0]
		listExpr = call.Args[1]
	}

	fieldName, err := parseFieldIdent(fieldExpr)
	if err != nil {
		return atomicPredicate{}, err
	}

	value, err := parseLiteral(listExpr)
	if err != nil {
		return atomicPredicate{}, err
	}

	return atomicPredicate{Field: fieldName, Op: OpIN, Value: value}, nil
}

func parseStartsWith(call *exprpb.Expr_Call) (atomicPredicate, error) {
	var fieldExpr *exprpb.Expr
	var valueExpr *exprpb.Expr

	if call.Target != nil {
		if len(call.Args) != 1 {
			return atomicPredicate{}, errors.New("startsWith with receiver must have exactly one argument")
		}
		fieldExpr = call.Target
		valueExpr = call.Args[0]
	} else {
		if len(call.Args) != 2 {
			return atomicPredicate{}, errors.New("startsWith must have exactly two arguments")
		}
		fieldExpr = call.Args[0]
		valueExpr = call.Args[1]
	}

	fieldName, err := parseFieldIdent(fieldExpr)
	if err != nil {
		return atomicPredicate{}, err
	}

	value, err := parseLiteral(valueExpr)
	if err != nil {
		return atomicPredicate{}, err
	}

	str, ok := value.(string)
	if !ok {
		return atomicPredicate{}, errors.New("startsWith requires a string literal argument")
	}

	return atomicPredicate{Field: fieldName, Op: OpSW, Value: str}, nil
}

func parseFieldIdent(expr *exprpb.Expr) (string, error) {
	ident := expr.GetIdentExpr()
	if ident == nil {
		return "", errors.New("left-hand side must be an identifier")
	}
	return ident.GetName(), nil
}

func parseLiteral(expr *exprpb.Expr) (any, error) {
	if constant := expr.GetConstExpr(); constant != nil {
		switch constant.ConstantKind.(type) {
		case *exprpb.Constant_StringValue:
			return constant.GetStringValue(), nil
		case *exprpb.Constant_Int64Value:
			return float64(constant.GetInt64Value()), nil
		case *exprpb.Constant_Uint64Value:
			return float64(constant.GetUint64Value()), nil
		case *exprpb.Constant_DoubleValue:
			return constant.GetDoubleValue(), nil
		default:
			return nil, fmt.Errorf("literal type %T is not supported", constant.ConstantKind)
		}
	}

	if list := expr.GetListExpr(); list != nil {
		elements := list.GetElements()
		values := make([]string, len(elements))
		for i, elem := range elements {
			val, err := parseLiteral(elem)
			if err != nil {
				return nil, fmt.Errorf("list literal element %d: %w", i, err)
			}
			str, ok := val.(string)
			if !ok {
				return nil, errors.New("list literal elements must be strings")
			}
			values[i] = str
		}
		return values, nil
	}

	if call := expr.GetCallExpr(); call != nil && call.Function == "timestamp" {
		if call.Target != nil || len(call.Args) != 1 {
			return nil, errors.New("timestamp() expects a single string argument")
		}
		arg := call.Args[0].GetConstExpr()
		if arg == nil {
			return nil, errors.New("timestamp() argument must be a string literal")
		}
		str := arg.GetStringValue()
		if str == "" {
			return time.Time{}, errors.New("timestamp() argument must not be empty")
		}

		if t, err := time.Parse(time.RFC3339Nano, str); err == nil {
			return t, nil
		} else if t, err := time.Parse(time.RFC3339, str); err == nil {
			return t, nil
		} else {
			return nil, fmt.Errorf("timestamp literal %q is not RFC3339", str)
		}
	}

	return nil, errors.New("right-hand side must be a literal, list literal, or timestamp() call")
}

func validateLiteral(kind ValueKind, op Op, value any) error {
	switch kind {
	case KindString:
		switch op {
		case OpIN:
			list, ok := value.([]string)
			if !ok {
				return fmt.Errorf("expected list of %s literals", kind)
			}
			if len(list) == 0 {
				return errors.New("list literal must not be empty")
			}
			for _, item := range list {
				if item == "" {
					return errors.New("list literal must not contain empty strings")
				}
			}
		default:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("expected %s literal", kind)
			}
		}
	case KindNumber:
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("expected %s literal", kind)
		}
	case KindTimestamp:
		if _, ok := value.(time.Time); !ok {
			return fmt.Errorf("expected %s literal", kind)
		}
	default:
		return fmt.Errorf("unsupported field kind %s", kind)
	}
	return nil
}

func callSetter(setter SetterFunc, field reflect.Value, value any) error {
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setter(field, value)
	}
	return setter(field, value)
}

func assignValue(field reflect.Value, value any) error {
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return assignValue(field.Elem(), value)
	}

	if field.Kind() == reflect.Interface {
		field.Set(reflect.ValueOf(value))
		return nil
	}

	switch v := value.(type) {
	case string:
		if field.Kind() != reflect.String {
			return fmt.Errorf("expected string-compatible destination, got %s", field.Kind())
		}
		field.SetString(v)
	case []string:
		if field.Kind() != reflect.Slice {
			return fmt.Errorf("expected slice destination, got %s", field.Kind())
		}
		if field.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("expected slice of strings destination, got %s", field.Type().Elem().Kind())
		}
		clone := make([]string, len(v))
		copy(clone, v)
		field.Set(reflect.ValueOf(clone))
	case float64:
		return assignNumeric(field, v)
	case time.Time:
		if field.Type() != timeType {
			return fmt.Errorf("expected time.Time destination, got %s", field.Type())
		}
		field.Set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("unsupported literal type %T", value)
	}

	return nil
}

func assignNumeric(field reflect.Value, value float64) error {
	switch field.Kind() {
	case reflect.Float32, reflect.Float64:
		field.SetFloat(value)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if math.Trunc(value) != value {
			return fmt.Errorf("cannot assign non-integer value %v to integer field", value)
		}
		bits := field.Type().Bits()
		min := -1 << (bits - 1)
		max := (1 << (bits - 1)) - 1
		if value < float64(min) || value > float64(max) {
			return fmt.Errorf("value %v overflows integer field", value)
		}
		field.SetInt(int64(value))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if math.Trunc(value) != value {
			return fmt.Errorf("cannot assign non-integer value %v to unsigned integer field", value)
		}
		if value < 0 {
			return fmt.Errorf("cannot assign negative value %v to unsigned integer field", value)
		}
		bits := field.Type().Bits()
		max := (uint64(1) << bits) - 1
		if value > float64(max) {
			return fmt.Errorf("value %v overflows unsigned integer field", value)
		}
		field.SetUint(uint64(value))
		return nil
	default:
		return fmt.Errorf("numeric assignment requires integer or float field, got %s", field.Kind())
	}
}

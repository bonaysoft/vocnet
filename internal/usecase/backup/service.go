package backup

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
	"github.com/eslsoft/vocnet/internal/infrastructure/database/ent/migrate"
	_ "github.com/lib/pq"           // ensure postgres driver available
	_ "github.com/mattn/go-sqlite3" // ensure sqlite driver available
)

const (
	defaultBatchSize = 512
	formatVersion    = 1
)

var errNoTablesSelected = errors.New("backup: no tables selected")

type ProgressReporter interface {
	StartTable(table string, total int)
	Increment(table string, delta int)
	FinishTable(table string)
}

type noopProgress struct{}

func (noopProgress) StartTable(string, int) {}
func (noopProgress) Increment(string, int)  {}
func (noopProgress) FinishTable(string)     {}

type Service struct {
	driver     string
	dsn        string
	batchSize  int
	tables     []*schema.Table
	tableIndex map[string]*schema.Table
	schemaHash string
}

type Option func(*Service)

func WithBatchSize(size int) Option {
	return func(s *Service) {
		if size > 0 {
			s.batchSize = size
		}
	}
}

// NewService constructs a backup service bound to the provided database driver and DSN.
func NewService(driver, dsn string, opts ...Option) (*Service, error) {
	driver = strings.TrimSpace(strings.ToLower(driver))
	if driver == "" {
		return nil, errors.New("backup: driver is required")
	}
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("backup: DSN is required")
	}

	tables, err := schema.CopyTables(migrate.Tables)
	if err != nil {
		return nil, fmt.Errorf("copy ent schema tables: %w", err)
	}
	tableIndex := make(map[string]*schema.Table, len(tables))
	for _, tbl := range tables {
		tableIndex[tbl.Name] = tbl
	}

	svc := &Service{
		driver:     driver,
		dsn:        dsn,
		batchSize:  defaultBatchSize,
		tables:     tables,
		tableIndex: tableIndex,
		schemaHash: computeSchemaHash(tables),
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc, nil
}

type ExportOption func(*exportConfig)

type exportConfig struct {
	tables   []string
	reporter ProgressReporter
}

// WithTables restricts export to the provided table names (snake_case as in DB).
func WithTables(tables []string) ExportOption {
	return func(cfg *exportConfig) {
		if len(tables) == 0 {
			return
		}
		cfg.tables = append([]string{}, tables...)
	}
}

// WithProgressReporter registers a reporter that receives progress callbacks during export.
func WithProgressReporter(reporter ProgressReporter) ExportOption {
	return func(cfg *exportConfig) {
		cfg.reporter = reporter
	}
}

type ImportOption func(*importConfig)

type importConfig struct {
	tables []string
}

// WithImportTables restricts import to the provided table names.
func WithImportTables(tables []string) ImportOption {
	return func(cfg *importConfig) {
		if len(tables) == 0 {
			return
		}
		cfg.tables = append([]string{}, tables...)
	}
}

type record struct {
	Type          string         `json:"type"`
	Version       int            `json:"version,omitempty"`
	ExportedAt    *time.Time     `json:"exported_at,omitempty"`
	EntSchemaHash string         `json:"ent_schema_hash,omitempty"`
	Tables        []string       `json:"tables,omitempty"`
	RowCounts     map[string]int `json:"row_counts,omitempty"`
	Payload       any            `json:"payload,omitempty"`
}

type rawRecord struct {
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	ExportedAt    *time.Time      `json:"exported_at"`
	EntSchemaHash string          `json:"ent_schema_hash"`
	Tables        []string        `json:"tables"`
	RowCounts     map[string]int  `json:"row_counts"`
	Payload       json.RawMessage `json:"payload"`
}

type sequenceKey struct {
	Table  string
	Column string
}

type sequenceStats map[sequenceKey]int64

func (s *Service) Export(ctx context.Context, w io.Writer, opts ...ExportOption) error {
	cfg := exportConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	tables, err := s.selectTables(cfg.tables)
	if err != nil {
		return err
	}
	reporter := cfg.reporter
	if reporter == nil {
		reporter = noopProgress{}
	}

	db, err := s.openDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	counts := make(map[string]int, len(tables))
	for _, tbl := range tables {
		count, err := s.countTableRows(ctx, db, tbl.Name)
		if err != nil {
			return fmt.Errorf("count table %s: %w", tbl.Name, err)
		}
		counts[tbl.Name] = count
	}

	writer := bufio.NewWriter(w)
	defer writer.Flush()

	now := time.Now().UTC()
	meta := record{
		Type:          "meta",
		Version:       formatVersion,
		ExportedAt:    &now,
		EntSchemaHash: s.schemaHash,
		Tables:        tableNames(tables),
		RowCounts:     counts,
	}
	if err := writeRecord(writer, meta); err != nil {
		return err
	}

	for _, tbl := range tables {
		total := counts[tbl.Name]
		reporter.StartTable(tbl.Name, total)
		if err := s.exportTable(ctx, db, tbl, reporter, writer); err != nil {
			return err
		}
		reporter.FinishTable(tbl.Name)
	}
	return writer.Flush()
}

func (s *Service) Import(ctx context.Context, r io.Reader, opts ...ImportOption) error {
	cfg := importConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	tables, err := s.selectTables(cfg.tables)
	if err != nil {
		return err
	}
	tableFilter := make(map[string]*schema.Table, len(tables))
	for _, tbl := range tables {
		tableFilter[tbl.Name] = tbl
	}

	db, err := s.openDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	commit := false
	defer func() {
		if !commit {
			_ = tx.Rollback()
		}
	}()

	br := bufio.NewReader(r)
	var (
		metaSeen bool
		meta     rawRecord
		stats    = make(sequenceStats)
	)

	for {
		line, err := br.ReadBytes('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read backup: %w", err)
		}
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			var rec rawRecord
			if err := json.Unmarshal(line, &rec); err != nil {
				return fmt.Errorf("decode record: %w", err)
			}

			switch rec.Type {
			case "meta":
				metaSeen = true
				meta = rec
			default:
				tbl, ok := tableFilter[rec.Type]
				if !ok {
					// Skip records for tables not requested.
					break
				}
				if len(rec.Payload) == 0 {
					return fmt.Errorf("backup: missing payload for table %s", rec.Type)
				}
				if err := s.importRow(ctx, tx, tbl, rec.Payload, stats); err != nil {
					return err
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}

	if !metaSeen {
		return errors.New("backup: missing meta record")
	}
	if meta.Version != formatVersion {
		return fmt.Errorf("backup: unsupported format version %d", meta.Version)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	commit = true

	if err := s.syncSequences(ctx, db, stats); err != nil {
		return err
	}
	return nil
}

func (s *Service) exportTable(ctx context.Context, db *sql.DB, table *schema.Table, reporter ProgressReporter, w io.Writer) error {
	columns := columnNames(table)
	if len(columns) == 0 {
		return nil
	}
	orderBy := buildOrderByClause(table)
	batch := s.batchSize
	if batch <= 0 {
		batch = defaultBatchSize
	}

	for offset := 0; ; offset += batch {
		query := fmt.Sprintf("SELECT %s FROM %s%s LIMIT %d OFFSET %d",
			strings.Join(columns, ", "),
			table.Name,
			orderBy,
			batch,
			offset,
		)
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("query %s: %w", table.Name, err)
		}

		rowCount := 0
		for rows.Next() {
			values := make([]any, len(columns))
			dest := make([]any, len(columns))
			for i := range dest {
				dest[i] = &values[i]
			}
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				return fmt.Errorf("scan %s: %w", table.Name, err)
			}
			rowMap, err := s.convertRow(table, columns, values)
			if err != nil {
				rows.Close()
				return err
			}
			if err := writeRecord(w, record{Type: table.Name, Payload: rowMap}); err != nil {
				rows.Close()
				return err
			}
			reporter.Increment(table.Name, 1)
			rowCount++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate %s: %w", table.Name, err)
		}
		rows.Close()
		if rowCount < batch {
			break
		}
	}
	return nil
}

func (s *Service) importRow(ctx context.Context, tx *sql.Tx, table *schema.Table, payload json.RawMessage, stats sequenceStats) error {
	values, err := decodePayload(table, payload)
	if err != nil {
		return fmt.Errorf("decode payload for %s: %w", table.Name, err)
	}
	if len(values) == 0 {
		return nil
	}

	cols := make([]string, 0, len(values))
	args := make([]any, 0, len(values))
	incrementCols := make(map[string]*schema.Column)
	for _, col := range table.Columns {
		val, ok := values[col.Name]
		if !ok {
			continue
		}
		if val == nil && !col.Nullable {
			if def, ok := defaultValueForColumn(col); ok {
				val = def
			} else if col.Default != nil {
				val = col.Default
			} else {
				return fmt.Errorf("backup: missing required value for %s.%s", table.Name, col.Name)
			}
		}
		cols = append(cols, col.Name)
		args = append(args, val)
		if col.Increment {
			incrementCols[col.Name] = col
		}
	}

	if len(cols) == 0 {
		return nil
	}

	placeholder := buildPlaceholders(s.driver, len(cols))
	if len(placeholder) != len(cols) {
		return fmt.Errorf("unsupported driver %q for placeholders", s.driver)
	}

	insert := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table.Name,
		strings.Join(cols, ", "),
		strings.Join(placeholder, ", "),
	)

	upsert, err := buildUpsertClause(s.driver, table, cols)
	if err != nil {
		return err
	}
	query := insert + upsert

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("insert into %s: %w", table.Name, err)
	}

	for colName := range incrementCols {
		if val, ok := values[colName]; ok {
			if max, ok := tryToInt64(val); ok {
				key := sequenceKey{Table: table.Name, Column: colName}
				if max > stats[key] {
					stats[key] = max
				}
			}
		}
	}
	return nil
}

func (s *Service) selectTables(requested []string) ([]*schema.Table, error) {
	if len(requested) == 0 {
		// Return tables sorted by name for deterministic order.
		tbls := make([]*schema.Table, len(s.tables))
		copy(tbls, s.tables)
		sort.Slice(tbls, func(i, j int) bool { return tbls[i].Name < tbls[j].Name })
		return tbls, nil
	}
	set := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		n := strings.TrimSpace(strings.ToLower(name))
		if n == "" {
			continue
		}
		if _, ok := s.tableIndex[n]; !ok {
			return nil, fmt.Errorf("backup: unsupported table %q", name)
		}
		set[n] = struct{}{}
	}
	if len(set) == 0 {
		return nil, errNoTablesSelected
	}
	tbls := make([]*schema.Table, 0, len(set))
	for _, tbl := range s.tables {
		if _, ok := set[tbl.Name]; ok {
			tbls = append(tbls, tbl)
		}
	}
	sort.Slice(tbls, func(i, j int) bool { return tbls[i].Name < tbls[j].Name })
	return tbls, nil
}

func (s *Service) openDB(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open(s.driver, s.dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if s.driver == "sqlite3" || s.driver == "sqlite" {
		if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
			db.Close()
			return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
		}
	}
	return db, nil
}

func (s *Service) countTableRows(ctx context.Context, db *sql.DB, table string) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	var count int
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) convertRow(table *schema.Table, columns []string, values []any) (map[string]any, error) {
	result := make(map[string]any, len(columns))
	for idx, name := range columns {
		colInfo := findColumn(table, name)
		if colInfo == nil {
			return nil, fmt.Errorf("column %s not found in table %s", name, table.Name)
		}
		val, err := convertDBValue(colInfo, values[idx])
		if err != nil {
			return nil, fmt.Errorf("convert %s.%s: %w", table.Name, name, err)
		}
		result[name] = val
	}
	return result, nil
}

func convertDBValue(col *schema.Column, value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case []byte:
		if col.Type == field.TypeBytes {
			if len(v) == 0 {
				return "", nil
			}
			return base64.StdEncoding.EncodeToString(v), nil
		}
		// database/sql often returns []byte for text columns.
		if col.Type == field.TypeJSON {
			cp := make(json.RawMessage, len(v))
			copy(cp, v)
			return cp, nil
		}
		return string(v), nil
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano), nil
	}

	switch col.Type {
	case field.TypeBool:
		switch vv := value.(type) {
		case bool:
			return vv, nil
		case int64:
			return vv != 0, nil
		case uint64:
			return vv != 0, nil
		default:
			return toBool(value)
		}
	case field.TypeInt8, field.TypeInt16, field.TypeInt32, field.TypeInt, field.TypeInt64:
		return toInt64(value)
	case field.TypeUint8, field.TypeUint16, field.TypeUint32, field.TypeUint, field.TypeUint64:
		return toUint64(value)
	case field.TypeFloat32, field.TypeFloat64:
		return toFloat64(value)
	default:
		return value, nil
	}
}

func decodePayload(table *schema.Table, payload json.RawMessage) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	var raw map[string]any
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	result := make(map[string]any, len(raw))
	for key, val := range raw {
		col := findColumn(table, key)
		if col == nil {
			return nil, fmt.Errorf("column %s not found in table %s", key, table.Name)
		}
		converted, err := convertJSONValue(col, val)
		if err != nil {
			return nil, fmt.Errorf("convert %s.%s: %w", table.Name, key, err)
		}
		result[key] = converted
	}
	return result, nil
}

func convertJSONValue(col *schema.Column, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch col.Type {
	case field.TypeBool:
		return toBool(value)
	case field.TypeInt8, field.TypeInt16, field.TypeInt32, field.TypeInt, field.TypeInt64:
		return toInt64(value)
	case field.TypeUint8, field.TypeUint16, field.TypeUint32, field.TypeUint, field.TypeUint64:
		return toUint64(value)
	case field.TypeFloat32, field.TypeFloat64:
		return toFloat64(value)
	case field.TypeTime:
		str, err := toString(value)
		if err != nil {
			return nil, err
		}
		if str == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339Nano, str)
		if err != nil {
			return nil, err
		}
		return t.UTC(), nil
	case field.TypeBytes:
		str, err := toString(value)
		if err != nil {
			return nil, err
		}
		if str == "" {
			return []byte{}, nil
		}
		return base64.StdEncoding.DecodeString(str)
	case field.TypeJSON:
		b, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(b), nil
	default:
		return value, nil
	}
}

func buildPlaceholders(driver string, count int) []string {
	switch driver {
	case "postgres", "postgresql":
		holders := make([]string, count)
		for i := 0; i < count; i++ {
			holders[i] = fmt.Sprintf("$%d", i+1)
		}
		return holders
	case "mysql":
		fallthrough
	case "sqlite3", "sqlite":
		holders := make([]string, count)
		for i := 0; i < count; i++ {
			holders[i] = "?"
		}
		return holders
	default:
		return nil
	}
}

func buildUpsertClause(driver string, table *schema.Table, insertCols []string) (string, error) {
	conflictCols := conflictColumns(table)
	if len(conflictCols) == 0 {
		return "", nil
	}
	updateCols := difference(insertCols, conflictCols)

	switch driver {
	case "postgres", "postgresql":
		if len(updateCols) == 0 {
			return fmt.Sprintf(" ON CONFLICT (%s) DO NOTHING", strings.Join(conflictCols, ", ")), nil
		}
		assignments := make([]string, len(updateCols))
		for i, col := range updateCols {
			assignments[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
		}
		return fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s",
			strings.Join(conflictCols, ", "),
			strings.Join(assignments, ", "),
		), nil
	case "sqlite3", "sqlite":
		if len(updateCols) == 0 {
			return fmt.Sprintf(" ON CONFLICT (%s) DO NOTHING", strings.Join(conflictCols, ", ")), nil
		}
		assignments := make([]string, len(updateCols))
		for i, col := range updateCols {
			assignments[i] = fmt.Sprintf("%s = excluded.%s", col, col)
		}
		return fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s",
			strings.Join(conflictCols, ", "),
			strings.Join(assignments, ", "),
		), nil
	case "mysql":
		if len(updateCols) == 0 {
			updateCols = conflictCols[:1]
		}
		assignments := make([]string, len(updateCols))
		for i, col := range updateCols {
			assignments[i] = fmt.Sprintf("%s = VALUES(%s)", col, col)
		}
		return fmt.Sprintf(" ON DUPLICATE KEY UPDATE %s", strings.Join(assignments, ", ")), nil
	default:
		return "", fmt.Errorf("backup: unsupported driver %q for upsert", driver)
	}
}

func conflictColumns(table *schema.Table) []string {
	if len(table.PrimaryKey) > 0 {
		cols := make([]string, len(table.PrimaryKey))
		for i, col := range table.PrimaryKey {
			cols[i] = col.Name
		}
		return cols
	}
	for _, idx := range table.Indexes {
		if idx.Unique && len(idx.Columns) > 0 {
			cols := make([]string, len(idx.Columns))
			for i, col := range idx.Columns {
				cols[i] = col.Name
			}
			return cols
		}
	}
	return nil
}

func buildOrderByClause(table *schema.Table) string {
	var cols []string
	if len(table.PrimaryKey) > 0 {
		for _, col := range table.PrimaryKey {
			cols = append(cols, col.Name)
		}
	} else {
		for _, col := range table.Columns {
			cols = append(cols, col.Name)
		}
	}
	if len(cols) == 0 {
		return ""
	}
	return " ORDER BY " + strings.Join(cols, ", ")
}

func columnNames(table *schema.Table) []string {
	cols := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		cols[i] = col.Name
	}
	return cols
}

func tableNames(tables []*schema.Table) []string {
	names := make([]string, len(tables))
	for i, tbl := range tables {
		names[i] = tbl.Name
	}
	return names
}

func difference(slice []string, exclude []string) []string {
	set := make(map[string]struct{}, len(exclude))
	for _, item := range exclude {
		set[item] = struct{}{}
	}
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if _, ok := set[item]; !ok {
			result = append(result, item)
		}
	}
	return result
}

func findColumn(table *schema.Table, name string) *schema.Column {
	for _, col := range table.Columns {
		if col.Name == name {
			return col
		}
	}
	return nil
}

func computeSchemaHash(tables []*schema.Table) string {
	builder := &strings.Builder{}
	sortedTables := make([]*schema.Table, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool { return sortedTables[i].Name < sortedTables[j].Name })

	for _, tbl := range sortedTables {
		builder.WriteString(tbl.Name)
		builder.WriteString("|cols:")
		sortedCols := make([]*schema.Column, len(tbl.Columns))
		copy(sortedCols, tbl.Columns)
		sort.Slice(sortedCols, func(i, j int) bool { return sortedCols[i].Name < sortedCols[j].Name })
		for _, col := range sortedCols {
			builder.WriteString(fmt.Sprintf("%s:%d:%t:%t:%t;", col.Name, col.Type, col.Nullable, col.Unique, col.Increment))
		}
		builder.WriteString("|pk:")
		for _, pk := range tbl.PrimaryKey {
			builder.WriteString(pk.Name)
			builder.WriteByte(',')
		}
		builder.WriteString("|idx:")
		sortedIdx := make([]*schema.Index, len(tbl.Indexes))
		copy(sortedIdx, tbl.Indexes)
		sort.Slice(sortedIdx, func(i, j int) bool { return sortedIdx[i].Name < sortedIdx[j].Name })
		for _, idx := range sortedIdx {
			builder.WriteString(idx.Name)
			builder.WriteString(":")
			builder.WriteString(strconv.FormatBool(idx.Unique))
			builder.WriteString(":")
			for _, col := range idx.Columns {
				builder.WriteString(col.Name)
				builder.WriteByte(',')
			}
			builder.WriteByte(';')
		}
		builder.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return fmt.Sprintf("%x", sum[:])
}

func writeRecord(w io.Writer, rec record) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func (s *Service) syncSequences(ctx context.Context, db *sql.DB, stats sequenceStats) error {
	if len(stats) == 0 {
		return nil
	}
	if s.driver != "postgres" && s.driver != "postgresql" {
		return nil
	}
	for key, maxVal := range stats {
		if maxVal <= 0 {
			continue
		}
		query := fmt.Sprintf(
			"SELECT setval(pg_get_serial_sequence('%s', '%s'), GREATEST(%d, (SELECT COALESCE(MAX(%s), 0) FROM %s)))",
			key.Table,
			key.Column,
			maxVal,
			key.Column,
			key.Table,
		)
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("sync sequence for %s.%s: %w", key.Table, key.Column, err)
		}
	}
	return nil
}

func defaultValueForColumn(col *schema.Column) (any, bool) {
	switch col.Type {
	case field.TypeJSON:
		return json.RawMessage("[]"), true
	case field.TypeString:
		return "", true
	case field.TypeInt, field.TypeInt8, field.TypeInt16, field.TypeInt32, field.TypeInt64,
		field.TypeUint, field.TypeUint8, field.TypeUint16, field.TypeUint32, field.TypeUint64,
		field.TypeFloat32, field.TypeFloat64:
		return 0, true
	case field.TypeBool:
		return false, true
	default:
		return nil, false
	}
}

func tryToInt64(val any) (int64, bool) {
	switch v := val.(type) {
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case int:
		return int64(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, true
		}
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case uint64:
		if v > math.MaxInt64 {
			return math.MaxInt64, true
		}
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint:
		if uint64(v) > math.MaxInt64 {
			return math.MaxInt64, true
		}
		return int64(v), true
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return false, err
		}
		return i != 0, nil
	case string:
		switch strings.ToLower(v) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return false, fmt.Errorf("invalid bool value %q", v)
		}
	case float64:
		return v != 0, nil
	case float32:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int:
		return v != 0, nil
	default:
		return false, fmt.Errorf("unsupported bool type %T", value)
	}
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported int type %T", value)
	}
}

func toUint64(value any) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case uint32:
		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative integer %d for unsigned column", v)
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative integer %d for unsigned column", v)
		}
		return uint64(v), nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, err
		}
		if i < 0 {
			return 0, fmt.Errorf("negative integer %d for unsigned column", i)
		}
		return uint64(i), nil
	case string:
		return strconv.ParseUint(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported uint type %T", value)
	}
}

func toFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case json.Number:
		return v.Float64()
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unsupported float type %T", value)
	}
}

func toString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case json.Number:
		return v.String(), nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

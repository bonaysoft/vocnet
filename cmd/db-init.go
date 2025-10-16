/*
Copyright © 2025 Ambor <saltbo@foxmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/eslsoft/vocnet/internal/infrastructure/database"
	entdb "github.com/eslsoft/vocnet/internal/infrastructure/database/ent"
	"github.com/eslsoft/vocnet/internal/infrastructure/database/ent/word"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

// dbInitCmd initializes database schema then imports ECDICT data into new 'words' table
var dbInitCmd = &cobra.Command{
	Use:   "db-init",
	Short: "初始化数据库并导入词库",
	Long:  "执行数据库迁移并从 ECDICT 导入词库。注意: go-sqlite3 需要 CGO_ENABLED=1 构建。如需仅迁移不导入，可使用 --schema-only。",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		batch, _ := cmd.Flags().GetInt("batch")
		schemaOnly, _ := cmd.Flags().GetBool("schema-only")
		cacheDir, _ := cmd.Flags().GetString("cache-dir")
		noCache, _ := cmd.Flags().GetBool("no-cache")
		if err := runMigrations(); err != nil {
			return err
		}
		if schemaOnly {
			return nil
		}
		return importECDICT(cmd.Context(), url, batch, cacheDir, noCache)
	},
}

const (
	ecDictURL             = "https://github.com/skywind3000/ECDICT/releases/download/1.0.28/ecdict-sqlite-28.zip"
	maxUncompressedSQLite = 1000 << 20 // 1000 MiB safety guard against decompression bombs
	defaultBatchSize      = 1000
)

func safeUint64ToInt64(v uint64) (int64, error) {
	if v > math.MaxInt64 {
		return 0, fmt.Errorf("value %d exceeds int64 capacity", v)
	}
	return int64(v), nil
}

func init() {
	rootCmd.AddCommand(dbInitCmd)
	dbInitCmd.Flags().String("url", ecDictURL, "ECDICT 下载地址")
	dbInitCmd.Flags().Int("batch", defaultBatchSize, "批量插入大小")
	dbInitCmd.Flags().Bool("schema-only", false, "仅执行数据库迁移，不导入词库")
	dbInitCmd.Flags().String("cache-dir", "", "ECDICT 缓存目录 (默认: 用户缓存目录/vocnet)")
	dbInitCmd.Flags().Bool("no-cache", false, "忽略本地缓存, 强制重新下载")
}

type wordRecord struct {
	Word        string
	Phonetic    sql.NullString
	Definition  sql.NullString
	Pos         sql.NullString
	Translation sql.NullString
	Exchange    sql.NullString
	Tags        sql.NullString // retained but currently unused for words import
}

// inflection relation extracted from exchange field
type inflectionRel struct {
	Lemma string
	Type  string
}

func importECDICT(ctx context.Context, url string, batchSize int, cacheDirFlag string, noCache bool) error { //nolint:gocognit,gocyclo // orchestration pulls IO, decompression, and batching into one workflow
	start := time.Now()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("开始导入 ECDICT: %s", url)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "ecdict-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Resolve cache directory
	cacheDir, zipPath, fromCache, err := prepareCachePath(url, cacheDirFlag, noCache)
	if err != nil {
		return err
	}
	if !fromCache {
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return fmt.Errorf("创建缓存目录失败: %w", err)
		}
		log.Printf("下载 ECDICT 到缓存: %s", zipPath)
		if err := downloadFile(ctx, url, zipPath); err != nil {
			return err
		}
	} else {
		log.Printf("使用缓存文件: %s", zipPath)
	}
	sqlitePath, err := unzipSingle(func(name string) bool { return strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sqlite") }, zipPath, tmpDir)
	if err != nil {
		return err
	}
	log.Printf("已解压 sqlite: %s", sqlitePath)

	sqldb, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return err
	}
	defer sqldb.Close()

	// NOTE: ECDICT schema sample (stardict): word, phonetic, definition, translation, pos, collins, oxford, tag, bnc, frq, exchange, detail, audio
	// We pull translation, tag, exchange if present; tolerate missing columns via COALESCE where possible.
	rows, err := sqldb.QueryContext(ctx, `SELECT word, phonetic, definition, pos, translation, exchange, tag FROM stardict`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// First collect all records (we need a global map to know which words are inflections of which lemma)
	records := make([]wordRecord, 0, 500000)
	for rows.Next() {
		var r wordRecord
		if err := rows.Scan(&r.Word, &r.Phonetic, &r.Definition, &r.Pos, &r.Translation, &r.Exchange, &r.Tags); err != nil {
			return err
		}
		r.Word = strings.TrimSpace(r.Word)
		if r.Word == "" || !isSingleWord(r.Word) || isAllEmpty(r) {
			continue
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Build inflection map: word(lower) -> (lemma, type)
	inflectionMap := make(map[string]inflectionRel)
	for _, r := range records {
		exchange := strings.TrimSpace(nullStringVal(r.Exchange))
		if exchange == "" {
			continue
		}
		pairs := parseExchangePairs(exchange)
		for _, p := range pairs { // p.word is inflected form, p.code is normalized type
			// 忽略 code=lemma (0:root) 这种“指向原形”的反向信息，避免把真正的原形标成别人的变形
			if p.code == entity.WordTypeLemma {
				continue
			}
			lw := strings.ToLower(p.word)
			if lw == "" || lw == strings.ToLower(r.Word) {
				continue
			}
			// only set if not already set (first lemma wins)
			if _, exists := inflectionMap[lw]; !exists {
				inflectionMap[lw] = inflectionRel{Lemma: r.Word, Type: p.code}
			}
		}
	}

	entClient, cleanup, err := database.NewEntClient(cfg)
	if err != nil {
		return fmt.Errorf("连接目标数据库失败: %w", err)
	}
	defer cleanup()

	// quick sanity check to ensure table exists (gives clearer error than bulk insert)
	if _, err := entClient.Word.Query().Limit(1).All(ctx); err != nil {
		return fmt.Errorf("验证 words 表失败: %w", err)
	}

	// Batch insert with word_type & lemma resolution. Rules:
	// - If word itself appears as an inflection of some other lemma: word_type = that type, lemma = that lemma
	// - Else if it provides exchange forms (i.e., it acts as base), word_type='lemma', lemma=NULL
	// - Else word_type='lemma' (default)
	// Note: a word can be both a lemma and an inflection (e.g., "read" past==present). Prefer lemma (keep lemma row) so lookup returns meanings.
	total := 0
	batchStart := 0
	for batchStart < len(records) {
		end := batchStart + batchSize
		if end > len(records) {
			end = len(records)
		}
		if err := insertBatchEnt(ctx, entClient, records[batchStart:end], inflectionMap); err != nil {
			return err
		}
		total += (end - batchStart)
		log.Printf("已导入 %d", total)
		batchStart = end
	}
	log.Printf("导入完成: %d 条, 耗时 %s", total, time.Since(start))
	return nil
}

// helpers
func downloadFile(ctx context.Context, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: %s", resp.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func unzipSingle(match func(string) bool, zipPath, dstDir string) (string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if match(f.Name) {
			if f.UncompressedSize64 > maxUncompressedSQLite {
				return "", fmt.Errorf("uncompressed size %d exceeds safety limit", f.UncompressedSize64)
			}
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			outPath := filepath.Join(dstDir, filepath.Base(f.Name))
			out, err := os.Create(outPath)
			if err != nil {
				return "", err
			}
			size, err := safeUint64ToInt64(f.UncompressedSize64)
			if err != nil {
				out.Close()
				return "", err
			}
			written, err := io.CopyN(out, rc, size)
			if err != nil && !errors.Is(err, io.EOF) {
				out.Close()
				return "", err
			}
			if written != size {
				out.Close()
				return "", fmt.Errorf("unexpected truncated copy: wrote %d bytes of %d", written, f.UncompressedSize64)
			}
			out.Close()
			return outPath, nil
		}
	}
	return "", errors.New("zip 中未找到 sqlite 文件")
}

// ensureWordsTable 已移除：不再提供运行时自动建表，改为强制迁移。

// (legacy single-pass insert function removed)

func insertBatchEnt(ctx context.Context, client *entdb.Client, batch []wordRecord, inflectionMap map[string]inflectionRel) error {
	if len(batch) == 0 {
		return nil
	}
	builders := make([]*entdb.WordCreate, 0, len(batch))
	for _, w := range batch {
		meanings, err := buildMeanings(w)
		if err != nil {
			return fmt.Errorf("构建 %s 的释义失败: %w", w.Word, err)
		}
		phonetics := buildPhonetics(w.Phonetic)
		if len(meanings) == 0 && len(phonetics) == 0 {
			continue
		}
		wordType := entity.WordTypeLemma
		var lemmaPtr *string
		if rel, ok := inflectionMap[strings.ToLower(w.Word)]; ok {
			if !strings.EqualFold(rel.Lemma, w.Word) {
				wordType = rel.Type
				lemmaPtr = &rel.Lemma
			}
		}
		builder := client.Word.Create().
			SetText(w.Word).
			SetLanguage("en").
			SetWordType(wordType).
			SetNillableLemma(lemmaPtr)
		if len(phonetics) > 0 {
			builder.SetPhonetics(phonetics)
		}
		if len(meanings) > 0 {
			builder.SetDefinitions(meanings)
		}
		if tags := buildTags(w.Tags); len(tags) > 0 {
			builder.SetCategories(tags)
		}
		builders = append(builders, builder)
	}
	if len(builders) == 0 {
		return nil
	}
	return client.Word.CreateBulk(builders...).
		OnConflictColumns(word.FieldLanguage, word.FieldText, word.FieldWordType).
		UpdateNewValues().
		Exec(ctx)
}

func buildTags(ns sql.NullString) []string {
	if !ns.Valid {
		return nil
	}
	s := strings.TrimSpace(ns.String)
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(parts))
	ordered := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		ordered = append(ordered, p)
	}
	if len(ordered) == 0 {
		return nil
	}
	return ordered
}

func buildPhonetics(ns sql.NullString) []entity.WordPhonetic {
	if !ns.Valid {
		return nil
	}
	ipa := strings.TrimSpace(ns.String)
	if ipa == "" {
		return nil
	}
	return []entity.WordPhonetic{
		{IPA: ipa, Dialect: "en-US"},
	}
}

// buildMeanings converts record fields into structured meanings for ent.
func buildMeanings(w wordRecord) ([]entity.WordDefinition, error) {
	defLines := splitLines(nullStringVal(w.Definition))
	transLines := splitLines(nullStringVal(w.Translation))
	if len(defLines) == 0 && len(transLines) == 0 {
		return nil, nil
	}

	// 已放弃 weight：不再解析 w.Pos 中的数字权重，仅依赖每行行首的 POS 标记。

	// 2. 从释义/翻译行里提取行首的词性标记 (如 vt., vi., n., v., adj.)
	type agg struct {
		pos   string
		defs  []string
		trans []string
	}
	groups := []*agg{}

	// Definitions: capture lines
	for _, line := range defLines {
		pos, rest := extractLeadingPOS(line)
		// we don't try to merge definitions of same pos; always new group
		groups = append(groups, &agg{pos: pos, defs: []string{rest}})
	}
	// Translations: appended after all definitions, keep independent
	for _, line := range transLines {
		pos, rest := extractLeadingPOS(line)
		groups = append(groups, &agg{pos: pos, trans: []string{rest}})
	}

	if len(groups) == 0 {
		return nil, nil
	}

	// 改为：每一条原始行 -> 一条 meaning，不再合并同 POS 行。
	type lineMeaning struct {
		pos  string
		lang entity.Language
		text string
	}
	var lm []lineMeaning
	for _, g := range groups {
		for _, d := range g.defs {
			lm = append(lm, lineMeaning{pos: g.pos, lang: entity.LanguageEnglish, text: d})
		}
		for _, tr := range g.trans {
			lm = append(lm, lineMeaning{pos: g.pos, lang: entity.LanguageChinese, text: tr})
		}
	}
	if len(lm) == 0 {
		return nil, nil
	}

	// 构建 meanings: 逐行转换，无权重。
	meaningsSlice := make([]entity.WordDefinition, 0, len(lm))
	for _, it := range lm {
		text := strings.TrimSpace(it.text)
		if text == "" {
			continue
		}
		lang := entity.NormalizeLanguage(it.lang)
		meaningsSlice = append(meaningsSlice, entity.WordDefinition{
			Pos:      strings.TrimSpace(it.pos),
			Text:     text,
			Language: lang,
		})
	}
	if len(meaningsSlice) == 0 {
		return nil, nil
	}
	return meaningsSlice, nil
}

// extractLeadingPOS 尝试解析行首词性标记，返回 (pos, 剩余文本)。若没有匹配返回 pos=""。
func extractLeadingPOS(line string) (string, string) {
	s := strings.TrimSpace(line)
	if s == "" {
		return "", ""
	}
	lower := strings.ToLower(s)
	// 候选列表按长度排序，先匹配更长的 (vt, vi 在 v 之前)
	candidates := []string{"vt", "vi", "adj", "adv", "prep", "pron", "conj", "interj", "int", "num", "art", "aux", "abbr", "pref", "suf", "noun", "n", "v"}
	for _, cand := range candidates {
		matchLen := len(cand)
		if len(lower) < matchLen {
			continue
		}
		if strings.HasPrefix(lower, cand) {
			rest := s[matchLen:]
			if rest == "" {
				// 完整行只包含候选字符串，不视为标记
				break
			}
			next := rest[0]
			if next != '.' && next != ' ' && next != '\t' {
				continue
			}
			// 跳过可选的 '.' 以及随后的空白
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "."))
			pos := normalizePOSWithDot(cand)
			return pos, rest
		}
	}
	return "", s
}

func normalizePOSWithDot(pos string) string {
	if pos == "noun" {
		pos = "n"
	}
	return pos + "."
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

// (weight parsing removed)

// Exchange parsing restored (classification only, no extra rows inserted)
type exchangePair struct{ code, word string }

func parseExchangePairs(s string) []exchangePair {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "/")
	out := make([]exchangePair, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		code := "other"
		val := part
		if left, right, ok := strings.Cut(part, ":"); ok {
			code = left
			val = right
		}
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		norm := normalizeExchangeCode(code)
		key := norm + "|" + strings.ToLower(val)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, exchangePair{code: norm, word: val})
	}
	return out
}

// normalizeExchangeCode maps ECDICT exchange codes to word_type strings.
// Rule (finalized): Only these three get abbreviations: past participle -> pp, present participle -> ing, third person singular -> 3sg.
// Others keep readable full words without underscores.
// Mapping:
//
//	p -> past
//	d -> pp            (past participle)
//	i -> ing           (present participle / gerund)
//	3 -> 3sg           (third person singular present)
//	r -> comparative
//	t -> superlative
//	s -> plural
//	0 -> lemma
//	1 -> variant
//
// Unrecognized codes returned unchanged (may be treated as "other").
func normalizeExchangeCode(c string) string {
	switch c {
	case "p":
		return "past"
	case "d":
		return "pp"
	case "i":
		return "ing"
	case "3":
		return "3sg"
	case "r":
		return "comparative"
	case "t":
		return "superlative"
	case "s":
		return "plural"
	case "0":
		return entity.WordTypeLemma
	case "1":
		return "variant"
	default:
		return c
	}
}

func isSingleWord(w string) bool {
	if strings.ContainsAny(w, " \t\n") {
		return false
	}
	// Exclude obvious multi-item constructs containing commas or semicolons
	if strings.ContainsAny(w, ",;") {
		return false
	}
	// Basic sanity: allow hyphenated and apostrophes
	return true
}

func isAllEmpty(r wordRecord) bool {
	return strings.TrimSpace(nullStringVal(r.Phonetic)) == "" &&
		strings.TrimSpace(nullStringVal(r.Definition)) == "" &&
		strings.TrimSpace(nullStringVal(r.Pos)) == "" &&
		strings.TrimSpace(nullStringVal(r.Translation)) == "" &&
		strings.TrimSpace(nullStringVal(r.Exchange)) == ""
}

func nullStringVal(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// runMigrations applies ent-managed schema migrations to the target database.
func runMigrations() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	client, cleanup, err := database.NewEntClient(cfg)
	if err != nil {
		return fmt.Errorf("创建 ent 客户端失败: %w", err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("执行 ent 迁移失败: %w", err)
	}

	dsn, err := cfg.DatabaseURL()
	if err != nil {
		return fmt.Errorf("解析数据库 DSN 失败: %w", err)
	}

	if err := ensurePostgresJSONTags(ctx, dsn); err != nil {
		return fmt.Errorf("升级 tags 列到 jsonb 失败: %w", err)
	}

	log.Println("数据库迁移完成")
	return nil
}

func ensurePostgresJSONTags(ctx context.Context, dsn string) error {
	if !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		return nil
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	const q = `SELECT udt_name FROM information_schema.columns WHERE table_name = 'words' AND column_name = 'tags'`
	var udt sql.NullString
	if err := db.QueryRowContext(ctx, q).Scan(&udt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if !udt.Valid {
		return nil
	}
	if udt.String == "jsonb" {
		return nil
	}
	if udt.String != "text[]" {
		return nil
	}

	_, err = db.ExecContext(ctx, `ALTER TABLE words ALTER COLUMN tags TYPE jsonb USING to_jsonb(tags);
		ALTER TABLE words ALTER COLUMN tags SET DEFAULT '[]'::jsonb;`)
	return err
}

// prepareCachePath decides cache location and returns (cacheDir, zipPath, fromCache, error)
func prepareCachePath(url, cacheDirFlag string, noCache bool) (string, string, bool, error) {
	// Determine base cache dir
	var base string
	if cacheDirFlag != "" {
		base = cacheDirFlag
	} else {
		userCache, err := os.UserCacheDir()
		if err != nil {
			return "", "", false, fmt.Errorf("获取用户缓存目录失败: %w", err)
		}
		base = filepath.Join(userCache, "vocnet")
	}
	// stable filename from URL hash
	h := crc32.ChecksumIEEE([]byte(url))
	name := fmt.Sprintf("ecdict-%08x.zip", h)
	zipPath := filepath.Join(base, name)
	if !noCache {
		if st, err := os.Stat(zipPath); err == nil && st.Size() > 0 {
			return base, zipPath, true, nil
		}
	}
	return base, zipPath, false, nil
}

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
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/eslsoft/vocnet/internal/infrastructure/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

const ecDictURL = "https://github.com/skywind3000/ECDICT/releases/download/1.0.28/ecdict-sqlite-28.zip"

func init() {
	rootCmd.AddCommand(dbInitCmd)
	dbInitCmd.Flags().String("url", ecDictURL, "ECDICT 下载地址")
	dbInitCmd.Flags().Int("batch", 1000, "批量插入大小")
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

// JSON marshal helpers for meanings & forms (aligned with entity.WordDefinition / VocForm but minimal).
type jsonMeaning struct {
	Pos      string `json:"pos"`
	Text     string `json:"text"`
	Language string `json:"language"`
}

// inflection relation extracted from exchange field
type inflectionRel struct {
	Lemma string
	Type  string
}

func importECDICT(ctx context.Context, url string, batchSize int, cacheDirFlag string, noCache bool) error {
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

	pgpool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		return err
	}
	defer pgpool.Close()

	// Ensure new words table exists (migration must have created it)
	if _, err := pgpool.Exec(ctx, "SELECT 1 FROM words LIMIT 1"); err != nil {
		return fmt.Errorf("words 表不存在或无法访问，请先执行迁移: %w", err)
	}

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
		if strings.TrimSpace(nullStringVal(r.Exchange)) == "" {
			continue
		}
		pairs := parseExchangePairs(nullStringVal(r.Exchange))
		for _, p := range pairs { // p.word is inflected form, p.code is normalized type
			// 忽略 code=lemma (0:root) 这种“指向原形”的反向信息，避免把真正的原形标成别人的变形
			if p.code == "lemma" {
				continue
			}
			lw := strings.ToLower(p.word)
			if lw == "" || lw == strings.ToLower(r.Word) {
				continue
			}
			// only set if not already set (first lemma wins)
			if _, exists := inflectionMap[lw]; !exists {
				inflectionMap[lw] = inflectionRel{Lemma: r.Word, Type: p.code}
			} else {
				// TODO: potential conflict (same inflected form claimed by multiple lemmas). We keep first; could log or collect stats.
			}
		}
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
		if err := insertBatchClassified(ctx, pgpool, records[batchStart:end], inflectionMap); err != nil {
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
			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				return "", err
			}
			out.Close()
			return outPath, nil
		}
	}
	return "", errors.New("zip 中未找到 sqlite 文件")
}

// ensureWordsTable 已移除：不再提供运行时自动建表，改为强制迁移。

// (legacy single-pass insert function removed)

// insertBatchClassified inserts records with lemma/word_type based on inflection map.
func insertBatchClassified(ctx context.Context, pool *pgxpool.Pool, batch []wordRecord, inflectionMap map[string]inflectionRel) error {
	if len(batch) == 0 {
		return nil
	}
	b := &pgx.Batch{}
	for _, w := range batch {
		meaningsJSON, err := buildMeanings(w)
		if err != nil {
			return fmt.Errorf("build meanings for %s: %w", w.Word, err)
		}
		phoneticsJSON := buildPhoneticsJSON(w.Phonetic)
		if meaningsJSON == nil && len(phoneticsJSON) == 0 {
			continue
		}
		// Determine word_type & lemma
		WordType := "lemma"
		var lemma any = nil
		if rel, ok := inflectionMap[strings.ToLower(w.Word)]; ok {
			// 已被别的 lemma 标记为某种形态 (不可能为 lemma，因为我们已跳过 code=lemma)
			if !strings.EqualFold(rel.Lemma, w.Word) {
				WordType = rel.Type
				lemma = rel.Lemma
			}
		}
		b.Queue(`INSERT INTO words (text, language, word_type, lemma, phonetics, meanings, tags)
				VALUES ($1,'en',$2,$3,COALESCE($4,'[]'::jsonb),COALESCE($5,'[]'::jsonb),$6)
				ON CONFLICT (language, text, word_type) DO UPDATE SET phonetics=EXCLUDED.phonetics, meanings=EXCLUDED.meanings, tags=EXCLUDED.tags`,
			w.Word, WordType, lemma, phoneticsJSON, meaningsJSON, buildTagsArray(w.Tags))
	}
	br := pool.SendBatch(ctx, b)
	for i := 0; i < b.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return err
		}
	}
	return br.Close()
}

func buildTagsArray(ns sql.NullString) any {
	// ECDICT 的 tag 字段是以空格分隔（有时还混有逗号），例如："cet4 cet6 ky toefl ielts gre"。
	// 之前实现只按逗号切分，导致整串被当成一个标签。这里改为：
	// 1. 将逗号统一替换为空格
	// 2. 按任意空白分词 (strings.Fields)
	// 3. 去重（保持首次出现顺序）
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

func buildPhoneticsJSON(ns sql.NullString) []byte {
	if !ns.Valid {
		return nil
	}
	ipa := strings.TrimSpace(ns.String)
	if ipa == "" {
		return nil
	}
	payload, err := json.Marshal([]entity.WordPhonetic{{IPA: ipa, Dialect: "en-US"}})
	if err != nil {
		return nil
	}
	return payload
}

// buildMeanings converts record fields to a JSONB value for meanings
func buildMeanings(w wordRecord) (meanings any, err error) {
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
	meaningsSlice := make([]jsonMeaning, 0, len(lm))
	for _, it := range lm {
		text := strings.TrimSpace(it.text)
		if text == "" {
			continue
		}
		lang := entity.NormalizeLanguage(it.lang)
		meaningsSlice = append(meaningsSlice, jsonMeaning{
			Pos:      strings.TrimSpace(it.pos),
			Text:     text,
			Language: lang.Code(),
		})
	}
	if len(meaningsSlice) == 0 {
		return nil, nil
	}
	b, e := json.Marshal(meaningsSlice)
	if e != nil {
		return nil, e
	}
	return b, nil
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
	switch pos {
	case "noun":
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
		if idx := strings.Index(part, ":"); idx >= 0 {
			code = part[:idx]
			val = part[idx+1:]
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
		return "lemma"
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

func nullString(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
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

	if err := ensurePostgresJSONTags(ctx, cfg.DatabaseURL()); err != nil {
		return fmt.Errorf("升级 tags 列到 jsonb 失败: %w", err)
	}

	log.Println("数据库迁移完成")
	return nil
}

func ensurePostgresJSONTags(ctx context.Context, dsn string) error {
	if !(strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://")) {
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

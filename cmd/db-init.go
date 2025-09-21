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
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// dbInitCmd initializes database schema (words table if missing) then imports ECDICT data
var dbInitCmd = &cobra.Command{
	Use:   "db-init",
	Short: "初始化数据库并导入词库",
	Long:  "执行数据库迁移并从 ECDICT 导入词库。注意: go-sqlite3 需要 CGO_ENABLED=1 构建。如需仅迁移不导入，可使用 --schema-only。",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		batch, _ := cmd.Flags().GetInt("batch")
		schemaOnly, _ := cmd.Flags().GetBool("schema-only")
		if err := runMigrations(); err != nil {
			return err
		}
		if schemaOnly {
			return nil
		}
		return importECDICT(cmd.Context(), url, batch)
	},
}

const ecDictURL = "https://github.com/skywind3000/ECDICT/releases/download/1.0.28/ecdict-sqlite-28.zip"

func init() {
	rootCmd.AddCommand(dbInitCmd)
	dbInitCmd.Flags().String("url", ecDictURL, "ECDICT 下载地址")
	dbInitCmd.Flags().Int("batch", 1000, "批量插入大小")
	dbInitCmd.Flags().Bool("schema-only", false, "仅执行数据库迁移，不导入词库")
}

type wordRecord struct {
	Word        string
	Phonetic    sql.NullString
	Definition  sql.NullString
	Pos         sql.NullString
	Translation sql.NullString
	Exchange    sql.NullString
	Tags        sql.NullString // comma separated tags; split later
}

func importECDICT(ctx context.Context, url string, batchSize int) error {
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

	zipPath := filepath.Join(tmpDir, "ecdict.zip")
	if err := downloadFile(ctx, url, zipPath); err != nil {
		return err
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

	// words 表必须由迁移创建；若不存在则直接失败
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

	batch := make([]wordRecord, 0, batchSize)
	total := 0
	for rows.Next() {
		var r wordRecord
		if err := rows.Scan(&r.Word, &r.Phonetic, &r.Definition, &r.Pos, &r.Translation, &r.Exchange, &r.Tags); err != nil {
			return err
		}
		r.Word = strings.TrimSpace(r.Word)
		if r.Word == "" {
			continue
		}
		batch = append(batch, r)
		if len(batch) >= batchSize {
			if err := insertBatch(ctx, pgpool, batch); err != nil {
				return err
			}
			total += len(batch)
			log.Printf("已导入 %d", total)
			batch = batch[:0]
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(batch) > 0 {
		if err := insertBatch(ctx, pgpool, batch); err != nil {
			return err
		}
		total += len(batch)
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

func insertBatch(ctx context.Context, pool *pgxpool.Pool, batch []wordRecord) error {
	if len(batch) == 0 {
		return nil
	}
	b := &pgx.Batch{}
	for _, w := range batch {
		var tagArray any
		if w.Tags.Valid {
			parts := strings.Split(w.Tags.String, ",")
			cleaned := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					cleaned = append(cleaned, p)
				}
			}
			if len(cleaned) > 0 {
				tagArray = cleaned
			}
		}
		b.Queue(`INSERT INTO words (lemma, language, phonetic, pos, definition, translation, exchange, tags) 
			VALUES ($1,'en',$2,$3,$4,$5,$6,$7)
			ON CONFLICT (lemma) DO UPDATE SET phonetic=EXCLUDED.phonetic, pos=EXCLUDED.pos, definition=EXCLUDED.definition,
				translation=EXCLUDED.translation, exchange=EXCLUDED.exchange, tags=EXCLUDED.tags`,
			w.Word, nullString(w.Phonetic), nullString(w.Pos), nullString(w.Definition), nullString(w.Translation), nullString(w.Exchange), tagArray)
	}
	br := pool.SendBatch(ctx, b)
	for range batch {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return err
		}
	}
	return br.Close()
}

func nullString(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// runMigrations executes SQL migrations in sql/migrations via golang-migrate if available.
func runMigrations() error {
	// Build source URL (local file path). We assume working directory is repo root.
	src := "file://sql/migrations"
	// Use DATABASE_URL env or config for connection; prefer same config loader for consistency.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	dbURL := cfg.DatabaseURL()
	m, err := migrate.New(src, dbURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	log.Println("数据库迁移完成")
	return nil
}

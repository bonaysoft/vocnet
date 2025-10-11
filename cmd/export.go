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
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/eslsoft/vocnet/internal/usecase/backup"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	exportOutputKey = "backup.export.output"
	exportGzipKey   = "backup.export.gzip"
	exportTablesKey = "backup.export.tables"
	exportBatchKey  = "backup.export.batch_size"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出数据库内容为 NDJSON 备份",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("加载配置失败: %w", err)
		}

		outputPath := viper.GetString(exportOutputKey)
		gzipEnabled := viper.GetBool(exportGzipKey)
		tableList := tablesFromConfig(exportTablesKey)
		batchSize := viper.GetInt(exportBatchKey)

		if outputPath == "" {
			outputPath = defaultExportFilename(gzipEnabled)
		}
		if !gzipEnabled && outputPath != "-" && strings.HasSuffix(strings.ToLower(outputPath), ".gz") {
			gzipEnabled = true
		}

		driver, err := cfg.DatabaseDriver()
		if err != nil {
			return fmt.Errorf("解析数据库驱动失败: %w", err)
		}
		dsn, err := cfg.DatabaseURL()
		if err != nil {
			return fmt.Errorf("解析数据库 DSN 失败: %w", err)
		}

		service, err := backup.NewService(
			driver,
			dsn,
			backup.WithBatchSize(batchSize),
		)
		if err != nil {
			return fmt.Errorf("创建备份服务失败: %w", err)
		}

		var (
			writer   = cmd.OutOrStdout()
			closeFns []func() error
		)

		if outputPath != "-" {
			if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
				return fmt.Errorf("创建输出目录失败: %w", err)
			}
			file, openErr := os.Create(outputPath)
			if openErr != nil {
				return fmt.Errorf("创建备份文件失败: %w", openErr)
			}
			writer = file
			closeFns = append(closeFns, file.Close)
		}

		if gzipEnabled {
			gz := gzip.NewWriter(writer)
			writer = gz
			closeFns = append([]func() error{gz.Close}, closeFns...)
		}

		defer func() {
			for _, closer := range closeFns {
				if cerr := closer(); cerr != nil && err == nil {
					err = cerr
				}
			}
		}()

		progress := newCLIProgress(cmd.ErrOrStderr())
		exportOpts := []backup.ExportOption{backup.WithProgressReporter(progress)}
		if len(tableList) > 0 {
			exportOpts = append(exportOpts, backup.WithTables(tableList))
		}

		if err := service.Export(ctx, writer, exportOpts...); err != nil {
			return fmt.Errorf("导出备份失败: %w", err)
		}

		if outputPath == "-" {
			cmd.Println("导出完成: 输出到标准输出")
		} else {
			cmd.Printf("导出完成: %s\n", outputPath)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringP("output", "o", "", "备份输出文件路径，使用 - 表示标准输出")
	exportCmd.Flags().Bool("gzip", false, "使用 gzip 压缩输出")
	exportCmd.Flags().StringSlice("tables", nil, "仅导出指定表，逗号分隔或重复指定")
	exportCmd.Flags().Int("batch-size", 0, "导出批处理大小 (默认 512)")

	bindExportConfig()
}

func defaultExportFilename(gzipEnabled bool) string {
	ts := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("vocnet-backup-%s.jsonl", ts)
	if gzipEnabled {
		filename += ".gz"
	}
	return filename
}

func bindExportConfig() {
	bindFlagToViper(exportOutputKey, exportCmd.Flags().Lookup("output"))
	bindFlagToViper(exportGzipKey, exportCmd.Flags().Lookup("gzip"))
	bindFlagToViper(exportTablesKey, exportCmd.Flags().Lookup("tables"))
	bindFlagToViper(exportBatchKey, exportCmd.Flags().Lookup("batch-size"))
}

type cliProgress struct {
	out         io.Writer
	totals      map[string]int
	counts      map[string]int
	lastPrinted map[string]int
	steps       map[string]int
}

func newCLIProgress(out io.Writer) *cliProgress {
	return &cliProgress{
		out:         out,
		totals:      make(map[string]int),
		counts:      make(map[string]int),
		lastPrinted: make(map[string]int),
		steps:       make(map[string]int),
	}
}

func (p *cliProgress) StartTable(table string, total int) {
	if total < 0 {
		total = 0
	}
	p.totals[table] = total
	p.counts[table] = 0
	p.lastPrinted[table] = 0
	p.steps[table] = progressStep(total)
	fmt.Fprintf(p.out, "开始导出 %s (共 %d 行)\n", table, total)
}

func (p *cliProgress) Increment(table string, delta int) {
	if delta <= 0 {
		return
	}
	current := p.counts[table] + delta
	p.counts[table] = current
	total := p.totals[table]
	step := p.steps[table]
	if step <= 0 {
		step = 1
	}
	last := p.lastPrinted[table]
	if current == total || last == 0 || current-last >= step {
		p.printProgress(table, current, total)
		p.lastPrinted[table] = current
	}
}

func (p *cliProgress) FinishTable(table string) {
	current := p.counts[table]
	total := p.totals[table]
	if current != p.lastPrinted[table] {
		p.printProgress(table, current, total)
	}
	if total > 0 {
		fmt.Fprintf(p.out, "完成导出 %s: %d/%d 行\n", table, current, total)
	} else {
		fmt.Fprintf(p.out, "完成导出 %s: %d 行\n", table, current)
	}
	delete(p.counts, table)
	delete(p.totals, table)
	delete(p.lastPrinted, table)
	delete(p.steps, table)
}

func (p *cliProgress) printProgress(table string, current, total int) {
	if total > 0 {
		fmt.Fprintf(p.out, "导出进度 %s: %d/%d\n", table, current, total)
	} else {
		fmt.Fprintf(p.out, "导出进度 %s: 已处理 %d 行\n", table, current)
	}
}

func progressStep(total int) int {
	if total <= 0 {
		return 1000
	}
	step := total / 20
	if step < 1 {
		step = 1
	}
	if step > 1000 {
		step = 1000
	}
	return step
}

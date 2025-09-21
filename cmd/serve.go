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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	infraDB "github.com/eslsoft/vocnet/internal/infrastructure/database"
	"github.com/eslsoft/vocnet/internal/infrastructure/server"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start gRPC + HTTP gateway server",
	RunE: func(cmd *cobra.Command, args []string) error {
		seedWord, _ := cmd.Flags().GetString("seed-word")
		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Logger
		logger := logrus.New()
		lvl, _ := logrus.ParseLevel(cfg.Log.Level)
		logger.SetLevel(lvl)
		if cfg.Log.Format == "text" {
			logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
		}

		// DB connection (pgx pool)
		pool, err := infraDB.NewConnection(cfg)
		if err != nil {
			return fmt.Errorf("db connect: %w", err)
		}
		defer pool.Close()

		if seedWord != "" {
			// naive upsert-like insert ignore conflicts
			_, err := pool.Exec(context.Background(), `INSERT INTO words(lemma, language) VALUES ($1,'en') ON CONFLICT (language, lemma) DO NOTHING`, seedWord)
			if err != nil {
				logger.Warnf("seed word failed: %v", err)
			} else {
				logger.Infof("seeded word: %s", seedWord)
			}
		}

		// Build server
		srv := server.NewServer(cfg, logger, pool)

		// Run gRPC & HTTP concurrently
		errCh := make(chan error, 2)
		go func() { errCh <- srv.StartGRPC() }()
		go func() { errCh <- srv.StartHTTP() }()

		// Graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-sigCh:
			logger.Infof("received signal: %s, shutting down", sig)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
			return nil
		case err := <-errCh:
			if err != nil {
				return err
			}
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().String("seed-word", "", "(dev) seed a word lemma before starting for quick lookup test")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

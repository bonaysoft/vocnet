package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func tablesFromConfig(key string) []string {
	return normalizeTables(viper.GetStringSlice(key))
}

func normalizeTables(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		result = append(result, strings.ToLower(name))
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func bindFlagToViper(key string, flag *pflag.Flag) {
	if flag == nil {
		return
	}
	cobra.CheckErr(viper.BindPFlag(key, flag))
}

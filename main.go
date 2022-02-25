package main

import (
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"runtime"
)

var conf Config

var rootCmd = &cobra.Command{
	Use:   "tpcls",
	Short: "Transparent proxy tool for Clash",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-05-04 15:02:01",
			PadLevelText:    true,
		})
		if runtime.GOOS != "linux" {
			return errors.New("only support linux system")
		}
		return nil
	},
}

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Fix transparent proxy",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

package main

import (
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"runtime"
)

var clashHome string
var clashConfig string
var clashUI string
var clashUIAddr string

var rootCmd = &cobra.Command{
	Use:   "tpclash",
	Short: "Transparent proxy tool for Clash",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS != "linux" {
			return errors.New("only support linux system")
		}
		return nil
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run tpclash",
	Run: func(cmd *cobra.Command, args []string) {
		fix()
		run()
	},
}

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Fix transparent proxy",
	Run: func(cmd *cobra.Command, args []string) {
		fix()
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean tpclash iptables and route config",
	Run: func(cmd *cobra.Command, args []string) {
		clean()
	},
}

func init() {
	// init logrus
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-05-04 15:02:01",
		PadLevelText:    true,
	})

	rootCmd.PersistentFlags().StringVarP(&clashHome, "home", "d", "/data/clash", "clash home dir")
	rootCmd.PersistentFlags().StringVarP(&clashConfig, "config", "c", "/etc/clash.yaml", "clash config path")
	rootCmd.PersistentFlags().StringVarP(&clashUI, "ui", "u", "official", "clash dashboard(official/yacd)")
	rootCmd.PersistentFlags().StringVarP(&clashUIAddr, "listen", "l", "0.0.0.0:9527", "clash dashboard listen address")

	rootCmd.AddCommand(fixCmd, runCmd, cleanCmd)
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

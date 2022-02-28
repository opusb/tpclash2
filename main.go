package main

import (
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var clashHome string
var clashConfig string
var clashUI string

var conf *Conf

var rootCmd = &cobra.Command{
	Use:   "tpclash",
	Short: "Transparent proxy tool for Clash",
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

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract embed files",
	Run: func(cmd *cobra.Command, args []string) {
		copyFiles()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&clashHome, "home", "d", "/data/clash", "clash home dir")
	rootCmd.PersistentFlags().StringVarP(&clashConfig, "config", "c", "/etc/clash.yaml", "clash config path")
	rootCmd.PersistentFlags().StringVarP(&clashUI, "ui", "u", "official", "clash dashboard(official/yacd)")

	rootCmd.AddCommand(fixCmd, runCmd, cleanCmd, extractCmd)
}

func main() {
	cobra.OnInitialize(func() {
		// init logrus
		logrus.SetLevel(logrus.InfoLevel)
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			PadLevelText:    true,
		})

		// os check
		if runtime.GOOS != "linux" {
			logrus.Fatal("only support linux system")
		}

		// init config
		viper.SetConfigFile(clashConfig)
		viper.SetEnvPrefix("TPCLASH")
		viper.AutomaticEnv()

		logrus.Info("[main] load clash config...")
		err := viper.ReadInConfig()
		if err != nil {
			logrus.Fatalf("failed to load config: %v", err)
		}
		conf, err = parseConf()
		if err != nil {
			logrus.Fatal(err)
		}

		// copy static files
		createUser()
		mkHomeDir()
		copyFiles()
	})
	cobra.CheckErr(rootCmd.Execute())
}

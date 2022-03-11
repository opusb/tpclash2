package main

import (
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var conf TPClashConf
var clashConf *ClashConf
var commit string

var rootCmd = &cobra.Command{
	Use:     "tpclash",
	Short:   "Transparent proxy tool for Clash",
	Version: commit,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run tpclash",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean tpclash iptables and route config",
	Run: func(cmd *cobra.Command, args []string) {
		cleanIPTables()
		cleanRoute()
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
	cobra.EnableCommandSorting = false
	cobra.OnInitialize(tpClashInit)

	rootCmd.PersistentFlags().StringVarP(&conf.ClashHome, "home", "d", "/data/clash", "clash home dir")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashConfig, "config", "c", "/etc/clash.yaml", "clash config path")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashUI, "ui", "u", "yacd", "clash dashboard(official/yacd)")
	rootCmd.PersistentFlags().StringSliceVar(&conf.HijackDNS, "hijack-dns", nil, "hijack the target DNS address (default \"0.0.0.0/0\")")
	rootCmd.PersistentFlags().BoolVar(&conf.MMDB, "mmdb", true, "extract Country.mmdb file")
	rootCmd.PersistentFlags().BoolVar(&conf.LocalProxy, "local-proxy", true, "enable local proxy")

	rootCmd.PersistentFlags().StringVar(&conf.TproxyMark, "tproxy-mark", defaultTproxyMark, "tproxy mark")
	rootCmd.PersistentFlags().StringVar(&conf.ClashUser, "clash-user", defaultClashUser, "clash runtime user")
	rootCmd.PersistentFlags().StringVar(&conf.DirectGroup, "direct-group", defaultDirectGroup, "skip transparent proxy group")

	_ = rootCmd.PersistentFlags().MarkHidden("tproxy-mark")
	_ = rootCmd.PersistentFlags().MarkHidden("clash-user")
	_ = rootCmd.PersistentFlags().MarkHidden("direct-group")

	rootCmd.AddCommand(runCmd, cleanCmd, extractCmd)
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

func tpClashInit() {
	// init logrus
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		PadLevelText:    true,
	})

	// os check
	if runtime.GOOS != "linux" {
		logrus.Fatal("only support linux system")
	}

	// set default hijack dns
	if conf.HijackDNS == nil {
		conf.HijackDNS = []string{"0.0.0.0/0"}
	}

	// init config
	viper.SetConfigFile(conf.ClashConfig)
	viper.SetEnvPrefix("TPCLASH")
	viper.AutomaticEnv()

	logrus.Info("[main] load clash config...")
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatalf("failed to load config: %v", err)
	}
	clashConf, err = parseConf()
	if err != nil {
		logrus.Fatal(err)
	}

	if clashConf.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// copy static files
	ensureUserAndGroup()
	mkHomeDir()
}

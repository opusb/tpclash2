package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	conf      TPClashConf
	clashConf ClashConf
	proxyMode ProxyMode
)

var commit string

var rootCmd = &cobra.Command{
	Use:     "tpclash",
	Short:   "Transparent proxy tool for Clash",
	Version: commit,
	Run:     func(cmd *cobra.Command, args []string) { run() },
}

func init() {
	cobra.EnableCommandSorting = false
	cobra.OnInitialize(tpClashInit)

	rootCmd.PersistentFlags().StringVarP(&conf.ProxyMode, "proxy-mode", "m", "ebpf", "clash proxy mode(tproxy|ebpf)")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashHome, "home", "d", "/data/clash", "clash home dir")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashConfig, "config", "c", "/etc/clash.yaml", "clash config path")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashUI, "ui", "u", "yacd", "clash dashboard(official/yacd)")
	rootCmd.PersistentFlags().BoolVar(&conf.LocalProxy, "local-proxy", true, "enable local proxy")
	rootCmd.PersistentFlags().BoolVar(&conf.Debug, "debug", false, "enable debug log")

	rootCmd.PersistentFlags().StringVar(&conf.TproxyMark, "tproxy-mark", defaultTproxyMark, "tproxy mark")
	rootCmd.PersistentFlags().StringVar(&conf.ClashUser, "clash-user", defaultClashUser, "clash runtime user")
	rootCmd.PersistentFlags().StringVar(&conf.DirectGroup, "direct-group", defaultDirectGroup, "skip tproxy group")
	rootCmd.PersistentFlags().StringSliceVar(&conf.HijackDNS, "hijack-dns", nil, "hijack the target DNS address (default \"0.0.0.0/0\")")
	rootCmd.PersistentFlags().StringVar(&conf.HijackIP, "hijack-ip", "", "hijack target IP traffic (all|IP_ADDRESS)")
	rootCmd.PersistentFlags().BoolVar(&conf.DisableExtract, "disable-extract", false, "disable extract files")

}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

func tpClashInit() {
	// init logrus
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		PadLevelText:    true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// set default hijack dns
	if conf.HijackDNS == nil {
		conf.HijackDNS = []string{"0.0.0.0/0"}
	}

	// load clash config
	viper.SetConfigFile(conf.ClashConfig)
	viper.SetEnvPrefix("TPCLASH")
	viper.AutomaticEnv()

	logrus.Info("[main] loading config...")
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatalf("failed to load clash config: %v", err)
	}

	if err = parseClashConf(); err != nil {
		logrus.Fatal(err)
	}

	if err = parseProxyMode(); err != nil {
		logrus.Fatal(err)
	}

	if clashConf.Debug || conf.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// copy static files
	ensureUserAndGroup()
	ensureClashFiles()
	ensureSysctl()
}

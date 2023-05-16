package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/irai/packet/fastlog"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/mritd/logrus"
)

var (
	conf        TPClashConf
	clashConf   *ClashConf
	proxyMode   ProxyMode
	arpHijacker *ARPHijacker
)

var printVer bool
var build string
var commit string
var version string
var clash string

var updateCh chan struct{}

var rootCmd = &cobra.Command{
	Use:   "tpclash",
	Short: "Transparent proxy tool for Clash",
	Run: func(_ *cobra.Command, _ []string) {
		if printVer {
			return
		}

		var err error

		logrus.Info("[main] starting tpclash...")

		cmd := exec.Command(filepath.Join(conf.ClashHome, clashBiName), "-f", conf.ClashConfig, "-d", conf.ClashHome, "-ext-ui", filepath.Join(conf.ClashHome, conf.ClashUI))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{
			AmbientCaps: []uintptr{CAP_NET_BIND_SERVICE, CAP_NET_ADMIN, CAP_NET_RAW},
		}

		logrus.Debugf("[clash] running cmds: %v", cmd.Args)

		parent, pCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer pCancel()
		if !conf.AutoExit {
			parent = context.Background()
		}

		ctx, cancel := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		defer cancel()
		if err = cmd.Start(); err != nil {
			logrus.Error(err)
			cancel()
		}
		if cmd.Process == nil {
			logrus.Errorf("failed to start clash process: %v", cmd.Args)
			cancel()
		}

		if err = proxyMode.EnableProxy(); err != nil {
			logrus.Fatalf("failed to enable proxy: %v", err)
		}

		if conf.HijackIP != nil {
			if err = arpHijacker.hijack(ctx); err != nil {
				logrus.Fatalf("failed to start arp hijack: %v", err)
			}
		}

		go func() {
			if updateCh == nil {
				return
			}

			for range updateCh {
				logrus.Info("[main] local config changed, clash reloading...")

				apiAddr := viper.GetString("external-controller")
				if apiAddr == "" {
					apiAddr = "127.0.0.1:9090"
				}
				secret := viper.GetString("secret")

				req, err := http.NewRequest("PUT", "http://"+apiAddr+"/configs", bytes.NewReader([]byte(fmt.Sprintf(`{"path": "%s"}`, conf.ClashConfig))))
				if err != nil {
					logrus.Errorf("failed to create reload req: %v", err)
					return
				}
				req.Header.Set("Authorization", "Bearer "+secret)
				cli := &http.Client{}

				resp, err := cli.Do(req)
				defer func() { _ = resp.Body.Close() }()
				if err != nil {
					logrus.Errorf("failed to reload config: %v", err)
				}
				if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
					logrus.Errorf("failed to reload config: status %d", resp.StatusCode)
				}

				logrus.Info("[main] local config reload success...")
			}
		}()

		<-time.After(3 * time.Second)
		logrus.Info("[main] 🍄 提莫队长正在待命...")

		<-ctx.Done()
		logrus.Info("[main] 🛑 TPClash 正在停止...")

		if err = proxyMode.DisableProxy(); err != nil {
			logrus.Error(err)
		}

		if cmd.Process != nil {
			if err = cmd.Process.Kill(); err != nil {
				logrus.Error(err)
			}
		}

		logrus.Info("[main] 🛑 TPClash 已关闭!")
	},
}

func init() {
	cobra.EnableCommandSorting = false
	cobra.OnInitialize(tpClashInit)

	rootCmd.PersistentFlags().StringVarP(&conf.ClashHome, "home", "d", "/data/clash", "clash home dir")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashConfig, "config", "c", "/etc/clash.yaml", "clash config local path or remote url")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashUI, "ui", "u", "yacd", "clash dashboard(official|yacd)")
	rootCmd.PersistentFlags().BoolVar(&conf.Debug, "debug", false, "enable debug log")

	rootCmd.PersistentFlags().DurationVarP(&conf.CheckInterval, "check-interval", "i", 30*time.Second, "remote config check interval")
	rootCmd.PersistentFlags().StringSliceVar(&conf.HttpHeader, "http-header", []string{}, "http header when requesting a remote config(key=value)")

	rootCmd.PersistentFlags().IPSliceVar(&conf.HijackIP, "hijack-ip", nil, "hijack target IP traffic")
	rootCmd.PersistentFlags().BoolVar(&conf.DisableExtract, "disable-extract", false, "disable extract files")
	rootCmd.PersistentFlags().BoolVar(&conf.AutoExit, "test", false, "run in test mode, exit automatically after 5 minutes")

	rootCmd.PersistentFlags().BoolVarP(&printVer, "version", "v", false, "version for tpclash")

}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

func tpClashInit() {
	if printVer {
		showVersion()
		return
	}

	// copy static files
	ensureClashFiles()
	ensureSysctl()

	// download remote config
	if strings.HasPrefix(conf.ClashConfig, "http://") ||
		strings.HasPrefix(conf.ClashConfig, "https://") {
		logrus.Info("[main] use remote config...")

		updateCh = make(chan struct{})
		remoteAddr := conf.ClashConfig
		conf.ClashConfig = filepath.Join(conf.ClashHome, clashRemoteConfig)

		go func() {
			req, err := http.NewRequest("GET", remoteAddr, nil)
			if err != nil {
				logrus.Fatalf("failed to create remote config req: %v", err)
			}

			for _, kv := range conf.HttpHeader {
				ss := strings.Split(kv, "=")
				if len(ss) != 2 {
					logrus.Fatalf("failed to parse http header: %s", kv)
				}
				req.Header.Set(ss[0], ss[1])
			}

			cli := &http.Client{Timeout: 10 * time.Second}

			c := time.Tick(conf.CheckInterval)
			for range c {
				logrus.Info("[main] checking remote config...")

				resp, err := cli.Do(req)
				if err != nil {
					logrus.Errorf("failed to download remote config: %v", err)
					continue
				}
				defer func() { _ = resp.Body.Close() }()

				if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
					logrus.Errorf("failed to get remote config: status code %d", resp.StatusCode)
					continue
				}

				var buf bytes.Buffer
				if _, err = io.Copy(&buf, resp.Body); err != nil {
					logrus.Fatalf("failed to copy resp: %v", err)
					continue
				}

				if _, err = os.Stat(conf.ClashConfig); err == nil {
					bs, err := os.ReadFile(conf.ClashConfig)
					if err != nil {
						logrus.Errorf("failed to read config file: %s: %v", conf.ClashConfig, err)
						continue
					}
					if string(bs) == buf.String() {
						logrus.Info("[main] remote file not change, skip...")
						continue
					}

					logrus.Info("[main] remote file changed, updating local config...")
					if err = os.WriteFile(conf.ClashConfig, buf.Bytes(), 0644); err != nil {
						logrus.Errorf("failed to save remote config: %v", err)
					}
					updateCh <- struct{}{}
				} else {
					logrus.Info("[main] local config not found, updating local config...")
					if err = os.WriteFile(conf.ClashConfig, buf.Bytes(), 0644); err != nil {
						logrus.Errorf("failed to save remote config: %v", err)
					}
					updateCh <- struct{}{}
				}
			}
		}()
	}

	if _, err := os.Stat(conf.ClashConfig); err != nil {
		logrus.Info("[main] waiting remote config downloaded...")
		<-updateCh
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

	if clashConf, err = ParseClashConf(); err != nil {
		logrus.Fatal(err)
	}

	if proxyMode, err = NewProxyMode(clashConf, &conf); err != nil {
		logrus.Fatal(err)
	}

	arpHijacker = NewARPHijacker(clashConf, &conf)

	if clashConf.Debug || conf.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		fastlog.DefaultIOWriter = io.Discard
	}
}

func showVersion() {
	fmt.Printf("%s\nVersion: %s\nBuild: %s\nClash Core: %s\nCommit: %s\n", logo, version, build, clash, commit)
}

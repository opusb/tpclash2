package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/mritd/logrus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var conf TPClashConf

var build string
var commit string
var version string
var clash string

var rootCmd = &cobra.Command{
	Use:   "tpclash",
	Short: "Transparent proxy tool for Clash",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("%s\nVersion: %s\nBuild: %s\nClash Core: %s\nCommit: %s\n\n", logo, version, build, clash, commit)

		if conf.PrintVersion {
			return
		}

		var err error
		if conf.Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		logrus.Info("[main] starting tpclash...")

		// Initialize signal control Context
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		defer cancel()

		// Configure Sysctl
		Sysctl()

		// Extract Clash executable and built-in configuration files
		ExtractFiles(&conf)

		// Watch config file
		updateCh := WatchConfig(ctx, &conf)

		// Wait for the first config to return
		clashConfStr := <-updateCh

		// Check clash config
		cc, err := CheckConfig(clashConfStr)
		if err != nil {
			logrus.Fatal(err)
		}

		// Copy remote or local clash config file to internal path
		clashConfPath := filepath.Join(conf.ClashHome, InternalConfigName)
		if err = os.WriteFile(clashConfPath, []byte(clashConfStr), 0644); err != nil {
			logrus.Fatalf("[main] failed to copy clash config: %v", err)
		}

		// Create child process
		clashBinPath := filepath.Join(conf.ClashHome, InternalClashBinName)
		clashUIPath := filepath.Join(conf.ClashHome, conf.ClashUI)
		cmd := exec.Command(clashBinPath, "-f", clashConfPath, "-d", conf.ClashHome, "-ext-ui", clashUIPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{
			AmbientCaps: []uintptr{CAP_NET_BIND_SERVICE, CAP_NET_ADMIN, CAP_NET_RAW},
		}
		logrus.Infof("[main] running cmds: %v", cmd.Args)

		if err = cmd.Start(); err != nil {
			logrus.Fatalf("[main] failed to start clash process: %v: %v", err, cmd.Args)
			cancel()
		}
		if cmd.Process == nil {
			cancel()
			logrus.Fatalf("[main] failed to start clash process: %v", cmd.Args)
		}

		if err = EnableDockerCompatible(); err != nil {
			logrus.Errorf("[main] failed enable docker compatible: %v", err)
		}

		// Watch clash config changes, and automatically reload the config
		go AutoReload(updateCh, clashConfPath)

		logrus.Info("[main] 🍄 提莫队长正在待命...")
		if conf.Test {
			logrus.Warn("[main] test mode enabled, tpclash will automatically exit after 5 minutes...")
			go func() {
				<-time.After(5 * time.Minute)
				cancel()
			}()
		}

		var containerMap map[string]string
		if conf.EnableTracing {
			logrus.Infof("[main] 🔪 永远不要忘记, 吾等为何而战...")
			containerMap, err = startTracing(ctx, conf, cc)
			if err != nil {
				logrus.Errorf("[main] ❌ tracing project deploy failed: %v", err)
			}
		}

		<-ctx.Done()
		logrus.Info("[main] 🛑 TPClash 正在停止...")
		if err = DisableDockerCompatible(); err != nil {
			logrus.Errorf("[main] failed disable docker compatible: %v", err)
		}

		if conf.EnableTracing && containerMap != nil {
			logrus.Infof("[main] 🔪 恐惧, 是万敌之首...")

			tracingStopCtx, tracingStopCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer tracingStopCancel()

			err = stopTracing(tracingStopCtx, containerMap)
			if err != nil {
				logrus.Errorf("[main] ❌ tracing project stop failed: %v", err)
			}
		}

		if cmd.Process != nil {
			if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
				logrus.Error(err)
			}
		}

		logrus.Info("[main] 🛑 TPClash 已关闭!")
	},
}

var encCmd = &cobra.Command{
	Use:   "enc FILENAME",
	Short: "Encrypt config file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			_ = cmd.Help()
			return
		}
		if conf.ConfigEncPassword == "" {
			logrus.Fatalf("[enc] configuration file encryption password cannot be empty")
		}

		plaintext, err := os.ReadFile(args[0])
		if err != nil {
			logrus.Fatalf("[enc] failed to read config file: %v", err)
		}

		ciphertext := Encrypt(plaintext, conf.ConfigEncPassword)
		if err = os.WriteFile(args[0]+".enc", ciphertext, 0644); err != nil {
			logrus.Fatalf("[enc] failed to write encrypted config file: %v", err)
		}

		logrus.Infof("[enc] encrypted file storage location %s", args[0]+".enc")
	},
}

var decCmd = &cobra.Command{
	Use:   "dec FILENAME",
	Short: "Decrypt config file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			_ = cmd.Help()
			return
		}
		if conf.ConfigEncPassword == "" {
			logrus.Fatalf("[dec] configuration file encryption password cannot be empty")
		}

		ciphertext, err := os.ReadFile(args[0])
		if err != nil {
			logrus.Fatalf("[dec] failed to read encrypted config file: %v", err)
		}

		plaintext, err := Decrypt(ciphertext, conf.ConfigEncPassword)
		if err != nil {
			logrus.Fatalf("[dec] failed to decrypt config file: %v", err)
		}

		if err = os.WriteFile(strings.TrimSuffix(args[0], ".enc"), plaintext, 0644); err != nil {
			logrus.Fatalf("[dec] failde to write config file: %v", err)
		}

		logrus.Infof("[enc] decrypted file storage location %s", strings.TrimSuffix(args[0], ".enc"))
	},
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install TPClash",
	Run: func(cmd *cobra.Command, args []string) {
		_, err := exec.LookPath("systemctl")
		if err != nil {
			logrus.Fatal("[install] the systemctl command was not found, your system may not be based on systemd")
		}

		var reinstall bool
		_, err = os.Stat(filepath.Join(systemdDir, "tpclash.service"))
		reinstall = err == nil

		exePath, err := os.Executable()
		if err != nil {
			logrus.Fatalf("[install] unable to get executable file path: %v", err)
		}

		err = os.MkdirAll(installDir, 0755)
		if err != nil {
			logrus.Fatalf("[install] failed to create directory: %v", err)
		}

		src, err := os.Open(exePath)
		if err != nil {
			logrus.Fatalf("[install] failed to open tpclash file: %v", err)
		}
		defer func() { _ = src.Close() }()

		dst, err := os.OpenFile(filepath.Join(installDir, "tpclash"), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0755)
		if err != nil {
			logrus.Fatalf("[install] failed to create executable file: %v", err)
		}
		defer func() { _ = dst.Close() }()

		_, err = io.Copy(dst, src)
		if err != nil {
			logrus.Fatalf("[install] failed to copy executable file: %v", err)
		}

		opts := ""
		if conf.Debug {
			opts += "--debug"
		}
		if conf.ClashHome != "" {
			opts += fmt.Sprintf(" %s %s", "--home", conf.ClashHome)
		}
		if conf.ClashConfig != "" {
			opts += fmt.Sprintf(" %s %s", "--config", conf.ClashConfig)
		}
		if conf.ClashUI != "" {
			opts += fmt.Sprintf(" %s %s", "--ui", conf.ClashUI)
		}
		if conf.CheckInterval > 0 {
			opts += fmt.Sprintf(" %s %s", "--check-interval", conf.CheckInterval.String())
		}
		if len(conf.HttpHeader) > 0 {
			for _, h := range conf.HttpHeader {
				opts += fmt.Sprintf(" %s '%s'", "--http-header", h)
			}
		}
		if conf.ConfigEncPassword != "" {
			opts += fmt.Sprintf(" %s %s", "--config-password", conf.ConfigEncPassword)
		}
		if conf.DisableExtract {
			opts += "--disable-extract"
		}

		err = os.WriteFile(filepath.Join(systemdDir, "tpclash.service"), []byte(fmt.Sprintf(systmedTpl, opts)), 0644)
		if err != nil {
			logrus.Fatalf("[install] failed to create systemd service: %v", err)
		}

		fmt.Print(installedMessage)
		if reinstall {
			fmt.Print(reinstallMessage)
		}
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall TPClash",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(uninstallMessage)
		time.Sleep(30 * time.Second)

		logrus.Warnf("[uninstall] remove --> %s", filepath.Join(installDir, "tpclash"))
		err := os.RemoveAll(filepath.Join(installDir, "tpclash"))
		if err != nil {
			logrus.Fatalf("[uninstall] failed to remove executable file: %v", err)
		}

		logrus.Warnf("[uninstall] remove --> %s", filepath.Join(systemdDir, "tpclash.service"))
		err = os.RemoveAll(filepath.Join(systemdDir, "tpclash.service"))
		if err != nil {
			logrus.Fatalf("[uninstall] failed to remove systemd service: %v", err)
		}

		fmt.Print(uninstalledMessage)
	},
}

func init() {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(encCmd, decCmd, installCmd, uninstallCmd)

	rootCmd.PersistentFlags().BoolVar(&conf.Debug, "debug", false, "enable debug log")
	rootCmd.PersistentFlags().BoolVar(&conf.Test, "test", false, "enable test mode, tpclash will automatically exit after 5 minutes")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashHome, "home", "d", "/data/clash", "clash home dir")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashConfig, "config", "c", "/etc/clash.yaml", "clash config local path or remote url")
	rootCmd.PersistentFlags().StringVarP(&conf.ClashUI, "ui", "u", "yacd", "clash dashboard(official|yacd)")
	rootCmd.PersistentFlags().DurationVarP(&conf.CheckInterval, "check-interval", "i", 120*time.Second, "remote config check interval")
	rootCmd.PersistentFlags().StringSliceVar(&conf.HttpHeader, "http-header", []string{}, "http header when requesting a remote config(key=value)")
	rootCmd.PersistentFlags().DurationVar(&conf.HttpTimeout, "http-timeout", 10*time.Second, "http request timeout when requesting a remote config")
	rootCmd.PersistentFlags().StringVar(&conf.ConfigEncPassword, "config-password", "", "the password for encrypting the config file")
	rootCmd.PersistentFlags().BoolVar(&conf.DisableExtract, "disable-extract", false, "disable extract files")
	rootCmd.PersistentFlags().BoolVar(&conf.EnableTracing, "enable-tracing", false, "auto deploy tracing dashboard")
	rootCmd.PersistentFlags().BoolVarP(&conf.PrintVersion, "version", "v", false, "version for tpclash")
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

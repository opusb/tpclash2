package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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
			opts += " --debug"
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
			opts += " --disable-extract"
		}
		if conf.EnableTracing {
			opts += " --enable-tracing"
		}
		if conf.AllowStandardDNSPort {
			opts += " --allow-standard-dns"
		}
		if conf.AutoFixMode != "" {
			opts += fmt.Sprintf(" %s %s", "--auto-fix", conf.AutoFixMode)
		}

		err = os.WriteFile(filepath.Join(systemdDir, "tpclash.service"), []byte(fmt.Sprintf(systemdTpl, opts)), 0644)
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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	semver "github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [VERSION]",
	Short: "upgrade TPClash",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var target *semver.Version

		if len(args) == 1 {
			target, err = semver.NewVersion(args[0])
			if err != nil {
				logrus.Fatalf("[upgrade] failed to parse version: %v", err)
			}
			logrus.Infof("[upgrade] upgrade to the specified version: v%s", target)
		} else {
			logrus.Info("[upgrade] check out the latest version from github...")
			resp, err := http.Get(githubLatestApi)
			if err != nil {
				logrus.Fatalf("[upgrade] failed to request github api: %v", err)
			}
			defer func() { _ = resp.Close }()

			var buf bytes.Buffer
			if _, err = io.Copy(&buf, resp.Body); err != nil {
				logrus.Fatalf("[upgrade] failed to read response: %v", err)
			}

			remoteVersion := &struct {
				Value string `json:"tag_name"`
			}{}
			if err = json.Unmarshal(buf.Bytes(), remoteVersion); err != nil {
				logrus.Fatalf("[upgrade] failed to unmarshal response: %v", err)
			}

			target, err = semver.NewVersion(remoteVersion.Value)
			if err != nil {
				logrus.Fatalf("[upgrade] failed to parse remote version: v%s: %v", remoteVersion, err)
			}
			logrus.Infof("[upgrade] upgrade to the latest version: v%s", target)
		}

		currentPath, err := os.Executable()
		if err != nil {
			logrus.Fatalf("[upgrade] failed to get current executable file path: %v", err)
		}

		tmpFile, err := os.OpenFile(filepath.Join(conf.ClashHome, fmt.Sprintf("tpclash.v%s", target)), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0755)
		if err != nil {
			logrus.Fatalf("[upgrade] failed to create temp file: %v", err)
		}
		defer func() {
			_ = tmpFile.Close()
		}()

		downAddr := fmt.Sprintf(githubUpgradeAddr, target, binName)
		if conf.UpgradeWithGhProxy {
			downAddr = ghProxyAddr + downAddr
		}
		logrus.Infof("[upgrade] start downloading file: %s", downAddr)

		binResp, err := http.Get(downAddr)
		if err != nil {
			logrus.Fatalf("[upgrade] failed to download new version: v%s: %v", target, err)
		}
		defer func() { _ = binResp.Close }()

		if _, err = io.Copy(tmpFile, binResp.Body); err != nil {
			logrus.Fatalf("[upgrade] failed to write temp file: %v", err)
		}

		if err = os.Rename(tmpFile.Name(), currentPath); err != nil {
			logrus.Fatalf("[upgrade] rename failed: %v", err)
		}

		fmt.Print(upgradedMessage)
	},
}

func init() {
	upgradeCmd.PersistentFlags().BoolVar(&conf.UpgradeWithGhProxy, "with-ghproxy", true, "use ghproxy.com to download upgrade files")
}

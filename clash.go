//go:build linux
// +build linux

package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

func run() {
	logrus.Info("[main] starting clash...")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	copyFiles()

	if err := applySysctl(); err != nil {
		logrus.Fatalf("Fix Sysctl Error: %s", err)
	}

	if err := applyRoute(); err != nil {
		logrus.Fatalf("Fix IP Route Error: %s", err)
	}

	if err := applyIPTables(); err != nil {
		logrus.Fatalf("Fix IPTables Error: %s", err)
	}

	u, err := user.Lookup(conf.ClashUser)
	if err != nil {
		logrus.Fatalf("failed to get tpclash user: %v", err)
	}

	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	cmds := []string{filepath.Join(conf.ClashHome, "xclash"), "-f", conf.ClashConfig, "-d", conf.ClashHome, "-ext-ui", filepath.Join(conf.ClashHome, conf.ClashUI)}
	logrus.Debugf("[clash] running cmds: %v", cmds)

	cmd := exec.Command(cmds[0], cmds[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
		AmbientCaps: []uintptr{CAP_NET_BIND_SERVICE, CAP_NET_ADMIN, CAP_NET_RAW},
	}

	if err = cmd.Start(); err != nil {
		logrus.Error(err)
		cancel()
	}

	<-time.After(3 * time.Second)
	logrus.Info("[main] 🍄 提莫队长正在待命...")
	<-ctx.Done()

	cleanIPTables()
	cleanRoute()

	if cmd.Process != nil {
		if err = cmd.Process.Kill(); err != nil {
			logrus.Error(err)
		}
	}

	logrus.Info("TPClash exit...")
}

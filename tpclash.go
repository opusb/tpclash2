package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

func run() {
	logrus.Info("[main] starting tpclash...")

	var err error
	if err = proxyMode.EnableForward(); err != nil {
		logrus.Fatalf("failed to enable proxy: %v", err)
	}

	uid, gid := getUserIDs(conf.ClashUser)
	cmd := exec.Command(filepath.Join(conf.ClashHome, "xclash"), "-f", conf.ClashConfig, "-d", conf.ClashHome, "-ext-ui", filepath.Join(conf.ClashHome, conf.ClashUI))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uid,
			Gid: gid,
		},
		AmbientCaps: []uintptr{CAP_NET_BIND_SERVICE, CAP_NET_ADMIN, CAP_NET_RAW},
	}

	logrus.Debugf("[clash] running cmds: %v", cmd.Args)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	if err = cmd.Start(); err != nil {
		logrus.Error(err)
		cancel()
	}
	if cmd.Process == nil {
		logrus.Errorf("failed to start clash process: %v", cmd.Args)
		cancel()
	}

	<-time.After(3 * time.Second)
	logrus.Info("[main] 🍄 提莫队长正在待命...")

	<-ctx.Done()
	logrus.Info("[main] 🛑 TPClash 正在停止...")

	if err = proxyMode.DisableForward(); err != nil {
		logrus.Error(err)
	}

	if cmd.Process != nil {
		if err = cmd.Process.Kill(); err != nil {
			logrus.Error(err)
		}
	}

	logrus.Info("[main] 🛑 TPClash 已关闭!")
}

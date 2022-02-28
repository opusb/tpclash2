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

	"github.com/sirupsen/logrus"
)

func run() {
	logrus.Info("starting clash...")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		u, err := user.Lookup(clashUser)
		if err != nil {
			logrus.Fatalf("failed to get tpclash user: %v", err)
		}

		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)

		cmd := exec.Command(filepath.Join(clashHome, "xclash"), "-f", clashConfig, "-d", clashHome, "-ext-ui", clashUI)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
			AmbientCaps: []uintptr{CAP_NET_BIND_SERVICE, CAP_NET_ADMIN, CAP_NET_RAW},
		}

		err = cmd.Run()
		select {
		case <-ctx.Done():
		default:
			logrus.Fatal(err)
		}
	}()

	<-ctx.Done()
	logrus.Info("TPClash exit...")
}

func fix() {
	if err := applySysctl(); err != nil {
		logrus.Fatalf("Fix Sysctl Error: %s", err)
	}

	if err := applyRoute(); err != nil {
		logrus.Fatalf("Fix IP Route Error: %s", err)
	}

	if err := applyIPTables(); err != nil {
		logrus.Fatalf("Fix IPTables Error: %s", err)
	}
}

func clean() {
	if err := cleanIPTables(); err != nil {
		logrus.Fatalf("Clean IPTables Error: %v", err)
	}

	if err := cleanRoute(); err != nil {
		logrus.Fatalf("Clean Route Error: %v", err)
	}
}

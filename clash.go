//go:build linux

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-iptables/iptables"

	"github.com/sirupsen/logrus"
)

func run() {
	logrus.Info("[main] starting clash...")

	var err error
	if err = enableProxy(); err != nil {
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

	<-time.After(3 * time.Second)
	logrus.Info("[main] 🍄 提莫队长正在待命...")
	<-ctx.Done()

	if err = disableProxy(); err != nil {
		logrus.Error(err)
	}

	if cmd.Process != nil {
		if err = cmd.Process.Kill(); err != nil {
			logrus.Error(err)
		}
	}

	logrus.Info("TPClash exit...")
}

func enableProxy() error {
	m, err := getProxyMode()
	if err != nil {
		return err
	}

	return process(m.addForward, m.addForwardDNS, m.addLocal, m.addLocalDNS, m.apply)
}

func disableProxy() error {
	m, err := getProxyMode()
	if err != nil {
		return err
	}

	return process(m.delForward, m.delForwardDNS, m.delLocal, m.delLocalDNS, m.clean)
}

func getProxyMode() (ProxyMode, error) {
	var m ProxyMode

	switch strings.ToLower(conf.ProxyMode) {
	case "tproxy":
		ip4, err := newIPTables(iptables.ProtocolIPv4)
		if err != nil {
			return nil, err
		}

		m = &tproxyMode{
			ins:  ip4,
			tpcc: &conf,
			cc:   clashConf,
		}
	case "tun":

	default:
		return nil, fmt.Errorf("unsupported proxy mode: %s", conf.ProxyMode)
	}

	return m, nil
}

func process(fns ...func() error) error {
	var err error
	for _, fn := range fns {
		if err = fn(); err != nil {
			return err
		}
	}
	return nil
}

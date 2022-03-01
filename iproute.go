package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

func applyRoute() error {
	logrus.Info("[route] add ip rules...")
	bs, err := exec.Command("ip", "rule", "add", "fwmark", tproxyMark, "lookup", tproxyMark).CombinedOutput()
	if err != nil && !strings.Contains(string(bs), "File exists") {
		return fmt.Errorf("failed to create ip rule: %s, %v", bs, err)
	}

	logrus.Info("[route] add ip routes...")
	bs, err = exec.Command("ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", tproxyMark).CombinedOutput()
	if err != nil && !strings.Contains(string(bs), "File exists") {
		return fmt.Errorf("failed to create ip route: %s, %v", bs, err)
	}

	return nil
}

func cleanRoute() {
	logrus.Info("[route] clean ip rules...")
	bs, err := exec.Command("ip", "rule", "del", "fwmark", tproxyMark, "table", tproxyMark).CombinedOutput()
	if err != nil {
		logrus.Warnf("failed to clean route: %s, %v", strings.TrimSpace(string(bs)), err)
	}

	logrus.Info("[route] clean ip routes...")
	bs, err = exec.Command("ip", "route", "del", "local", "0.0.0.0/0", "dev", "lo", "table", tproxyMark).CombinedOutput()
	if err != nil {
		logrus.Warnf("failed to clean route: %s, %v", strings.TrimSpace(string(bs)), err)
	}
}

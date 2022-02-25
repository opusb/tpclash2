package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
)

func fixRoute() error {
	logrus.Info("check ip routes...")

	_ = exec.Command("ip", "rule", "del", "fwmark", tproxyMark, "table", tproxyMark).Run()
	_ = exec.Command("ip", "route", "del", "local", "0.0.0.0/0", "dev", "lo", "table", tproxyMark).Run()

	err := exec.Command("ip", "rule", "add", "fwmark", tproxyMark, "lookup", tproxyMark).Run()
	if err != nil {
		return fmt.Errorf("failed to create ip rule: %w", err)
	}

	err = exec.Command("ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", tproxyMark).Run()
	if err != nil {
		return fmt.Errorf("failed to create ip route: %w", err)
	}

	return nil
}

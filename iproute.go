package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
)

func applyRoute() error {
	logrus.Info("[route] add ip rules...")
	err := exec.Command("ip", "rule", "add", "fwmark", tproxyMark, "lookup", tproxyMark).Run()
	if err != nil {
		return fmt.Errorf("failed to create ip rule: %w", err)
	}

	logrus.Info("[route] add ip routes...")
	err = exec.Command("ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", tproxyMark).Run()
	if err != nil {
		return fmt.Errorf("failed to create ip route: %w", err)
	}

	return nil
}

func cleanRoute() error {
	logrus.Info("[route] delete ip rules...")
	err := exec.Command("ip", "rule", "del", "fwmark", tproxyMark, "table", tproxyMark).Run()
	if err != nil {
		return fmt.Errorf("failed to clean route: %w", err)
	}

	logrus.Info("[route] delete ip routes...")
	err = exec.Command("ip", "route", "del", "local", "0.0.0.0/0", "dev", "lo", "table", tproxyMark).Run()
	if err != nil {
		return fmt.Errorf("failed to clean route: %w", err)
	}

	return nil
}

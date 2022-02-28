package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
)

func applyRoute() error {
	logrus.Info("[route] add ip rules...")
	bs, err := exec.Command("ip", "rule", "add", "fwmark", tproxyMark, "lookup", tproxyMark).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create ip rule: %s, %v", bs, err)
	}

	logrus.Info("[route] add ip routes...")
	bs, err = exec.Command("ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", tproxyMark).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create ip route: %s, %v", bs, err)
	}

	return nil
}

func cleanRoute() error {
	logrus.Info("[route] delete ip rules...")
	bs, err := exec.Command("ip", "rule", "del", "fwmark", tproxyMark, "table", tproxyMark).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clean route: %s, %v", bs, err)
	}

	logrus.Info("[route] delete ip routes...")
	bs, err = exec.Command("ip", "route", "del", "local", "0.0.0.0/0", "dev", "lo", "table", tproxyMark).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clean route: %s, %v", bs, err)
	}

	return nil
}

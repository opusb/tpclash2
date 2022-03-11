//go:build linux
// +build linux

package main

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
)

var privateCIDR = []string{
	"0.0.0.0/8",
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"224.0.0.0/4",
	"240.0.0.0/4",
}

func newIPTables(p iptables.Protocol) (*iptables.IPTables, error) {
	logrus.Debug("[iptables] creating ipv4 instance...")
	return iptables.New(iptables.IPFamily(p), iptables.Timeout(3))
}

func createChain(ins *iptables.IPTables, table, chain string) error {
	logrus.Debugf("[iptables] checking if chain %s exists in table %s...", chain, table)
	ok, err := ins.ChainExists(table, chain)
	if err != nil {
		return fmt.Errorf("failed to check chain: %s, table: %s, error: %v", chain, table, err)
	}
	if !ok {
		logrus.Debugf("[iptables] chian %s not found, creating...", chain)
		err = ins.NewChain(table, chain)
		if err != nil {
			return fmt.Errorf("failed to create chain: %s, error: %v", chain, err)
		}
	}
	return nil
}

func skipPrivateNetwork(ins *iptables.IPTables, table, chain string) error {
	logrus.Debugf("[iptables] checking chain %s/%s rules...", table, chain)
	for _, cidr := range privateCIDR {
		logrus.Debugf("[iptables] append private cidr %s rule to %s/%s...", cidr, table, chain)
		err := ins.AppendUnique(table, chain, "-d", cidr, "-j", actionReturn)
		if err != nil {
			return fmt.Errorf("failed to append private cidr rules: %v", err)
		}
	}
	return nil
}

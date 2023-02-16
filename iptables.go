package main

import (
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
)

func newIPTables(p iptables.Protocol) (*iptables.IPTables, error) {
	logrus.Debug("[iptables] creating ipv4 instance...")
	return iptables.New(iptables.IPFamily(p), iptables.Timeout(3))
}

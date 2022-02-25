package main

import (
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"strconv"

	clsconfig "github.com/Dreamacro/clash/config"
	clsconstant "github.com/Dreamacro/clash/constant"
)

var (
	localCIDR = []string{
		"0.0.0.0/8",
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"224.0.0.0/4",
		"240.0.0.0/4",
	}
)

func fixIPTables() error {

	/* Check Config */

	logrus.Infof("loading clash config...")
	cbs, err := ioutil.ReadFile(conf.Path)
	if err != nil {
		return fmt.Errorf("faile to load clash config: %w", err)
	}
	clsc, err := clsconfig.Parse(cbs)
	if err != nil {
		return fmt.Errorf("faile to load clash config: %w", err)
	}

	if clsc.DNS.EnhancedMode != clsconstant.DNSFakeIP {
		return fmt.Errorf("only support fake-ip dns mode")
	}

	if clsc.General.TProxyPort < 1 {
		return fmt.Errorf("tproxy port in clash configuration is missing(tproxy-port)")
	}

	dnsHost, dnsPort, err := net.SplitHostPort(clsc.DNS.Listen)
	if err != nil {
		return fmt.Errorf("failed to parse clash dns listen config: %w", err)
	}

	dport, err := strconv.Atoi(dnsPort)
	if err != nil {
		return fmt.Errorf("failed to parse clash dns listen config: %w", err)
	}

	if dport < 1 {
		return fmt.Errorf("dns port in clash configuration is missing(dns.listen)")
	}

	/* Create *iptables.IPTables */

	logrus.Debug("creating ipv4 iptables instance...")
	ip4, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4), iptables.Timeout(3))
	if err != nil {
		return fmt.Errorf("faile to create ipv4 iptables instance: %w", err)
	}

	/* Forward Local Network Rules */

	if err = chain(ip4, tableMangle, chainIP4); err != nil {
		return err
	}

	if err = localNetwork(ip4, tableMangle, chainIP4); err != nil {
		return err
	}

	/* Gateway Local Network Rules */

	if err = chain(ip4, tableMangle, chainIP4Local); err != nil {
		return err
	}

	if err = localNetwork(ip4, tableMangle, chainIP4Local); err != nil {
		return err
	}

	/* TProxy Rules */

	logrus.Info("checking tproxy rules...")
	err = ip4.AppendUnique(tableMangle, chainIP4, "-p", "tcp", "-j", actionTProxy, "--on-port", strconv.Itoa(clsc.General.TProxyPort), "--tproxy-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %w", err)
	}
	err = ip4.AppendUnique(tableMangle, chainIP4, "-p", "udp", "-j", actionTProxy, "--on-port", strconv.Itoa(clsc.General.TProxyPort), "--tproxy-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append udp trpoxy rules: %w", err)
	}
	err = ip4.AppendUnique(tableMangle, chainIP4Local, "-p", "tcp", "-j", actionMark, "--set-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %w", err)
	}
	err = ip4.AppendUnique(tableMangle, chainIP4Local, "-p", "udp", "-j", actionMark, "--set-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %w", err)
	}

	/* DNS Rules */

	if err = chain(ip4, tableNat, chainIP4DNS); err != nil {
		return err
	}

	logrus.Infof("checking chain %s/%s rules...", tableNat, chainIP4DNS)
	var dnsSpec []string
	if dnsHost != "" {
		dnsSpec = []string{"-p", "udp", "-d", dnsHost, "--dport", "53", "-j", actionRedirect, "--to", dnsPort}
	} else {
		dnsSpec = []string{"-p", "udp", "--dport", "53", "-j", actionRedirect, "--to", dnsPort}
	}
	err = ip4.AppendUnique(tableNat, chainIP4DNS, dnsSpec...)
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %w", err)
	}

	/* TPClash Output Rules */

	logrus.Info("checking tplcash output rules...")
	err = ip4.AppendUnique(tableMangle, chainOutput, "-p", "tcp", "-m", "owner", "--uid-owner", clashUser, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %w", err)
	}
	err = ip4.AppendUnique(tableMangle, chainOutput, "-p", "udp", "-m", "owner", "--uid-owner", clashUser, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %w", err)
	}

	/* Fix ICMP */

	logrus.Info("checking icmp fake rules...")
	err = ip4.AppendUnique(tableNat, chainPreRouting, "-p", "icmp", "-d", clsc.DNS.FakeIPRange.IPNet().String(), "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to append icmp fake rules: %w", err)
	}

	/* Apply Rules */

	logrus.Info("apply all rules...")
	err = ip4.DeleteIfExists(tableMangle, chainPreRouting, "-j", chainIP4)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %w", tableMangle, chainPreRouting, chainIP4, err)
	}
	err = ip4.DeleteIfExists(tableMangle, chainOutput, "-j", chainIP4Local)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %w", tableMangle, chainOutput, chainIP4Local, err)
	}

	err = ip4.Insert(tableMangle, chainPreRouting, 0, "-j", chainIP4)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %w", tableMangle, chainPreRouting, chainIP4, err)
	}
	err = ip4.Insert(tableMangle, chainOutput, 0, "-j", chainIP4Local)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %w", tableMangle, chainOutput, chainIP4Local, err)
	}

	return nil
}

func chain(ins *iptables.IPTables, table, chain string) error {
	logrus.Infof("checking if chain %s exists in table %s...", chain, table)
	ok, err := ins.ChainExists(table, chain)
	if err != nil {
		return fmt.Errorf("failed to check chain: %s, table: %s, error: %w", chain, table, err)
	}
	if !ok {
		logrus.Infof("chian %s not found, creating...", chain)
		err = ins.NewChain(table, chain)
		if err != nil {
			return fmt.Errorf("failed to create chain: %s, error: %w", chain, err)
		}
	}
	return nil
}

func localNetwork(ins *iptables.IPTables, table, chain string) error {
	logrus.Infof("checking chain %s/%s rules...", table, chain)
	for _, cidr := range localCIDR {
		logrus.Infof("append local cidr %s rule to %s/%s...", cidr, table, chain)
		err := ins.AppendUnique(table, chain, "-d", cidr, "-j", actionReturn)
		if err != nil {
			return fmt.Errorf("failed to append local cidr rules: %w", err)
		}
	}
	return nil
}

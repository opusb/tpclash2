package main

import (
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"strconv"

	clsconfig "github.com/Dreamacro/clash/config"
	clsconst "github.com/Dreamacro/clash/constant"
)

type iptablesConf struct {
	DNSHost     string
	DNSPort     string
	TProxyPort  string
	FakeIPRange string
}

var localCIDR = []string{
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

func parseDNS() (*iptablesConf, error) {
	logrus.Info("[iptables] loading clash config...")
	cbs, err := ioutil.ReadFile(clashConfig)
	if err != nil {
		return nil, fmt.Errorf("faile to load clash config: %w", err)
	}
	clsc, err := clsconfig.Parse(cbs)
	if err != nil {
		return nil, fmt.Errorf("faile to load clash config: %w", err)
	}

	if clsc.DNS.EnhancedMode != clsconst.DNSFakeIP {
		return nil, fmt.Errorf("only support fake-ip dns mode")
	}

	if clsc.General.TProxyPort < 1 {
		return nil, fmt.Errorf("tproxy port in clash config is missing(tproxy-port)")
	}

	dnsHost, dnsPort, err := net.SplitHostPort(clsc.DNS.Listen)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clash dns listen config: %w", err)
	}

	dport, err := strconv.Atoi(dnsPort)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clash dns listen config: %w", err)
	}

	if dport < 1 {
		return nil, fmt.Errorf("dns port in clash config is missing(dns.listen)")
	}

	return &iptablesConf{dnsHost, dnsPort, strconv.Itoa(clsc.General.TProxyPort), clsc.DNS.FakeIPRange.IPNet().String()}, nil
}

func applyIPTables() error {
	/* Parser DNS config */

	conf, err := parseDNS()
	if err != nil {
		return err
	}

	/* Create *iptables.IPTables */

	ip4, err := newIPTables(iptables.ProtocolIPv4)
	if err != nil {
		return fmt.Errorf("faile to create ipv4 iptables instance: %w", err)
	}

	/* Forward Local Network Rules */

	if err = createChain(ip4, tableMangle, chainIP4); err != nil {
		return err
	}

	if err = applyLocalNetwork(ip4, tableMangle, chainIP4); err != nil {
		return err
	}

	/* Gateway Local Network Rules */

	if err = createChain(ip4, tableMangle, chainIP4Local); err != nil {
		return err
	}

	if err = applyLocalNetwork(ip4, tableMangle, chainIP4Local); err != nil {
		return err
	}

	/* TProxy Rules */

	logrus.Info("[iptables] checking tproxy rules...")
	err = ip4.AppendUnique(tableMangle, chainIP4, "-p", "tcp", "-j", actionTProxy, "--on-port", conf.TProxyPort, "--tproxy-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %w", err)
	}
	err = ip4.AppendUnique(tableMangle, chainIP4, "-p", "udp", "-j", actionTProxy, "--on-port", conf.TProxyPort, "--tproxy-mark", tproxyMark)
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

	if err = createChain(ip4, tableNat, chainIP4DNS); err != nil {
		return err
	}

	logrus.Infof("[iptables] checking chain %s/%s rules...", tableNat, chainIP4DNS)
	var dnsSpec []string
	if conf.DNSHost != "" {
		dnsSpec = []string{"-p", "udp", "-d", conf.DNSHost, "--dport", "53", "-j", actionRedirect, "--to", conf.DNSPort}
	} else {
		dnsSpec = []string{"-p", "udp", "--dport", "53", "-j", actionRedirect, "--to", conf.DNSPort}
	}
	err = ip4.AppendUnique(tableNat, chainIP4DNS, dnsSpec...)
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %w", err)
	}

	/* TPClash Output Rules */

	logrus.Info("[iptables] checking tpclash output rules...")
	err = ip4.AppendUnique(tableMangle, chainOutput, "-p", "tcp", "-m", "owner", "--uid-owner", clashUser, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %w", err)
	}
	err = ip4.AppendUnique(tableMangle, chainOutput, "-p", "udp", "-m", "owner", "--uid-owner", clashUser, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %w", err)
	}

	/* Fix ICMP */

	logrus.Info("[iptables] checking icmp fake rules...")
	err = ip4.AppendUnique(tableNat, chainPreRouting, "-p", "icmp", "-d", conf.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to append icmp fake rules: %w", err)
	}

	/* Apply Rules */

	logrus.Info("[iptables] apply all rules...")
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

func createChain(ins *iptables.IPTables, table, chain string) error {
	logrus.Infof("[iptables] checking if chain %s exists in table %s...", chain, table)
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

func applyLocalNetwork(ins *iptables.IPTables, table, chain string) error {
	logrus.Infof("[iptables] checking chain %s/%s rules...", table, chain)
	for _, cidr := range localCIDR {
		logrus.Infof("append local cidr %s rule to %s/%s...", cidr, table, chain)
		err := ins.AppendUnique(table, chain, "-d", cidr, "-j", actionReturn)
		if err != nil {
			return fmt.Errorf("failed to append local cidr rules: %w", err)
		}
	}
	return nil
}

func cleanIPTables() error {
	logrus.Info("[iptables] clean rules...")

	/* Parser DNS config */

	conf, err := parseDNS()
	if err != nil {
		return err
	}

	/* Create *iptables.IPTables */

	ip4, err := newIPTables(iptables.ProtocolIPv4)
	if err != nil {
		return fmt.Errorf("faile to create ipv4 iptables instance: %w", err)
	}

	/* Clean iptables */

	logrus.Infof("[iptables] clean %s/%s...", tableMangle, chainPreRouting)
	err = ip4.DeleteIfExists(tableMangle, chainPreRouting, "-j", chainIP4)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %w", tableMangle, chainPreRouting, chainIP4, err)
	}

	logrus.Infof("[iptables] clean %s/%s...", tableMangle, chainOutput)
	err = ip4.DeleteIfExists(tableMangle, chainOutput, "-j", chainIP4Local)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %w", tableMangle, chainOutput, chainIP4Local, err)
	}

	logrus.Info("[iptables] clean icmp fake...")
	err = ip4.DeleteIfExists(tableNat, chainPreRouting, "-p", "icmp", "-d", conf.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to delete icmp fake rules: %w", err)
	}

	logrus.Info("[iptables] clean tpclash output...")
	err = ip4.DeleteIfExists(tableMangle, chainOutput, "-p", "tcp", "-m", "owner", "--uid-owner", clashUser, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to delete tpclash output rules: %w", err)
	}
	err = ip4.DeleteIfExists(tableMangle, chainOutput, "-p", "udp", "-m", "owner", "--uid-owner", clashUser, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to delete tpclash output rules: %w", err)
	}

	logrus.Info("[iptables] clean dns...")
	var dnsSpec []string
	if conf.DNSHost != "" {
		dnsSpec = []string{"-p", "udp", "-d", conf.DNSHost, "--dport", "53", "-j", actionRedirect, "--to", conf.DNSPort}
	} else {
		dnsSpec = []string{"-p", "udp", "--dport", "53", "-j", actionRedirect, "--to", conf.DNSPort}
	}
	err = ip4.DeleteIfExists(tableNat, chainIP4DNS, dnsSpec...)
	if err != nil {
		return fmt.Errorf("failed to delete dns rules: %w", err)
	}

	return nil
}

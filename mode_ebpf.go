package main

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"

	"github.com/sirupsen/logrus"
)

type ebpfMode struct {
	ins  *iptables.IPTables
	tpcc *TPClashConf
	cc   *ClashConf
}

func (m *ebpfMode) addForward() error { return nil }
func (m *ebpfMode) delForward() error { return nil }

func (m *ebpfMode) addForwardDNS() error {
	logrus.Debugf("[ebpf] add forward dns iptables rules...")

	var err error

	// iptables -t nat -N TP_CLASH_DNS_V4
	if err = createChain(m.ins, tableNat, chainIP4DNS); err != nil {
		return err
	}

	// iptables -t nat -A TP_CLASH_DNS_V4 -p udp -m udp --dst 0.0.0.0/0 --dport 53 -j REDIRECT --to-ports 1053
	for _, hDNS := range conf.HijackDNS {
		err = m.ins.AppendUnique(tableNat, chainIP4DNS, "-p", "udp", "--dst", hDNS, "--dport", "53", "-j", actionRedirect, "--to", m.cc.DNSPort)
		if err != nil {
			return fmt.Errorf("failed to append dns rules: %v", err)
		}
	}

	return nil
}
func (m *ebpfMode) delForwardDNS() error {
	logrus.Debugf("[ebpf] delete forward dns iptables rules...")

	ok, err := m.ins.ChainExists(tableNat, chainIP4DNS)
	if err != nil {
		return fmt.Errorf("failed to check chain %s/%s: %s", tableNat, chainIP4DNS, err)
	}
	if ok {
		err = m.ins.DeleteIfExists(tableNat, chainPreRouting, "-j", chainIP4DNS)
		if err != nil {
			return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableNat, chainPreRouting, chainIP4DNS, err)
		}
		err = m.ins.ClearAndDeleteChain(tableNat, chainIP4DNS)
		if err != nil {
			return fmt.Errorf("failed to delete chain: %s/%s, error: %v", tableNat, chainIP4DNS, err)
		}
	}

	return nil
}

func (m *ebpfMode) addLocal() error { return nil }
func (m *ebpfMode) delLocal() error { return nil }

func (m *ebpfMode) addLocalDNS() error { return nil }
func (m *ebpfMode) delLocalDNS() error { return nil }

func (m *ebpfMode) apply() error {
	logrus.Info("[iptables] apply all rules...")

	// iptables -t nat -A PREROUTING -j TP_CLASH_DNS_V4
	err := m.ins.AppendUnique(tableNat, chainPreRouting, "-j", chainIP4DNS)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableNat, chainPreRouting, chainIP4DNS, err)
	}

	return nil
}
func (m *ebpfMode) clean() error { return nil }

func (m *ebpfMode) EnableForward() error {
	return process(m.addForward, m.addForwardDNS, m.addLocal, m.addLocalDNS, m.apply)
}

func (m *ebpfMode) DisableForward() error {
	return process(m.delForward, m.delForwardDNS, m.delLocal, m.delLocalDNS, m.clean)
}

package main

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"

	"github.com/sirupsen/logrus"
)

type tunMode struct {
	ins  *iptables.IPTables
	tpcc *TPClashConf
	cc   *ClashConf
}

func (m *tunMode) addForward() error { return nil }
func (m *tunMode) delForward() error { return nil }

func (m *tunMode) addForwardDNS() error {
	logrus.Debugf("[tun] add forward dns iptables rules...")

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
func (m *tunMode) delForwardDNS() error {
	logrus.Debugf("[tun] delete forward dns iptables rules...")

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

func (m *tunMode) addLocal() error { return nil }
func (m *tunMode) delLocal() error { return nil }

func (m *tunMode) addLocalDNS() error {
	if !m.tpcc.LocalProxy {
		return nil
	}
	logrus.Debugf("[tun] add local dns iptables rules...")

	var err error

	// iptables -t nat -N TP_CLASH_DNS_LOCAL_V4
	if err = createChain(m.ins, tableNat, chainIP4DNSLocal); err != nil {
		return err
	}

	logrus.Debugf("[iptables] checking chain %s/%s rules...", tableNat, chainIP4DNSLocal)
	// iptables -t nat -A TP_CLASH_DNS_LOCAL_V4 -m owner --uid-owner m.tpcc.ClashUser -j RETURN
	err = m.ins.AppendUnique(tableNat, chainIP4DNSLocal, "-m", "owner", "--uid-owner", m.tpcc.ClashUser, "-j", actionReturn)

	// iptables -t nat -A TP_CLASH_DNS_LOCAL_V4 -m owner --gid-owner m.tpcc.DirectGroup -j RETURN
	err = m.ins.AppendUnique(tableNat, chainIP4DNSLocal, "-m", "owner", "--gid-owner", m.tpcc.DirectGroup, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway group skip rules: %v", err)
	}

	// iptables -t nat -A TP_CLASH_DNS_LOCAL_V4 -m owner --gid-owner systemd-resolve -j RETURN
	if checkGroup(systemdResolveGroup) {
		err = m.ins.AppendUnique(tableNat, chainIP4DNSLocal, "-m", "owner", "--gid-owner", systemdResolveGroup, "-j", actionReturn)
		if err != nil {
			return fmt.Errorf("failed to append gateway systemd-resolve skip rules: %v", err)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %v", err)
	}

	// iptables -t nat -A TP_CLASH_DNS_LOCAL_V4 -p udp -m udp -dst 0.0.0.0/0 --dport 53 -j REDIRECT --to-ports 1053
	for _, hDNS := range conf.HijackDNS {
		err = m.ins.AppendUnique(tableNat, chainIP4DNSLocal, "-p", "udp", "--dst", hDNS, "--dport", "53", "-j", actionRedirect, "--to", m.cc.DNSPort)
		if err != nil {
			return fmt.Errorf("failed to append dns rules: %v", err)
		}
	}

	return nil
}
func (m *tunMode) delLocalDNS() error {
	if !m.tpcc.LocalProxy {
		return nil
	}
	logrus.Debugf("[tun] delete local dns iptables rules...")

	ok, err := m.ins.ChainExists(tableNat, chainIP4DNSLocal)
	if err != nil {
		return fmt.Errorf("failed to check chain %s/%s: %s", tableNat, chainIP4DNSLocal, err)
	}
	if ok {
		err = m.ins.DeleteIfExists(tableNat, chainOutput, "-j", chainIP4DNSLocal)
		if err != nil {
			return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableNat, chainOutput, chainIP4DNSLocal, err)
		}
		err = m.ins.ClearAndDeleteChain(tableNat, chainIP4DNSLocal)
		if err != nil {
			return fmt.Errorf("failed to delete chain: %s/%s, error: %v", tableNat, chainIP4DNS, err)
		}
	}

	return nil
}

func (m *tunMode) addMisc() error {
	ok, err := m.ins.ChainExists(tableFilter, chainDockerUser)
	if err != nil {
		return fmt.Errorf("failed to check chain %s/%s: %s", tableFilter, chainDockerUser, err)
	}
	if ok {
		// iptables -t filter -I DOCKER-USER -j ACCEPT
		err = m.ins.Insert(tableFilter, chainDockerUser, 1, "-j", actionAccept)
		if err != nil {
			return fmt.Errorf("failed to append docker rules: %v", err)
		}
	}
	return nil
}
func (m *tunMode) delMisc() error {
	ok, err := m.ins.ChainExists(tableFilter, chainDockerUser)
	if err != nil {
		return nil
	}
	if ok {
		// iptables -t filter -I DOCKER-USER -j ACCEPT
		err = m.ins.DeleteIfExists(tableFilter, chainDockerUser, "-j", actionAccept)
		if err != nil {
			return fmt.Errorf("failed to delete docker rules: %v", err)
		}
	}
	return nil
}

func (m *tunMode) apply() error {
	logrus.Info("[iptables] apply all rules...")

	// iptables -t nat -A PREROUTING -j TP_CLASH_DNS_V4
	err := m.ins.AppendUnique(tableNat, chainPreRouting, "-j", chainIP4DNS)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableNat, chainPreRouting, chainIP4DNS, err)
	}

	if m.tpcc.LocalProxy {
		// iptables -t nat -A OUTPUT -j TP_CLASH_DNS_LOCAL_V4
		err = m.ins.AppendUnique(tableNat, chainOutput, "-j", chainIP4DNSLocal)
		if err != nil {
			return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableNat, chainOutput, chainIP4DNSLocal, err)
		}
	}

	return nil
}
func (m *tunMode) clean() error { return nil }

func (m *tunMode) EnableForward() error {
	return process(m.addForward, m.addForwardDNS, m.addLocal, m.addLocalDNS, m.addMisc, m.apply)
}

func (m *tunMode) DisableForward() error {
	return process(m.delForward, m.delForwardDNS, m.delLocal, m.delLocalDNS, m.delMisc, m.clean)
}

package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/coreos/go-iptables/iptables"
)

type tproxyMode struct {
	ins  *iptables.IPTables
	tpcc *TPClashConf
	cc   *ClashConf
}

func (m *tproxyMode) addForward() error {
	logrus.Debugf("[tproxy] add forward iptables rules...")

	var err error

	// iptables -t mangle -N TP_CLASH_V4
	if err = createChain(m.ins, tableMangle, chainIP4); err != nil {
		return err
	}

	// iptables -t mangle -A TP_CLASH_V4 -d 0.0.0.0/8 -j RETURN
	// iptables -t mangle -A TP_CLASH_V4 -d 127.0.0.0/8 -j RETURN
	// iptables -t mangle -A TP_CLASH_V4 -d 10.0.0.0/8 -j RETURN
	// iptables -t mangle -A TP_CLASH_V4 -d 172.16.0.0/12 -j RETURN
	// iptables -t mangle -A TP_CLASH_V4 -d 192.168.0.0/16 -j RETURN
	// iptables -t mangle -A TP_CLASH_V4 -d 169.254.0.0/16 -j RETURN
	// iptables -t mangle -A TP_CLASH_V4 -d 224.0.0.0/4 -j RETURN
	// iptables -t mangle -A TP_CLASH_V4 -d 240.0.0.0/4 -j RETURN
	if err = skipPrivateNetwork(m.ins, tableMangle, chainIP4); err != nil {
		return err
	}

	// iptables -t mangle -A TP_CLASH_V4 -p udp -m udp --dport 53 -j RETURN
	err = m.ins.AppendUnique(tableMangle, chainIP4, "-p", "udp", "--dport", "53", "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append forward dns skip rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_V4 -p udp -m udp --dport 123 -j RETURN
	err = m.ins.AppendUnique(tableMangle, chainIP4, "-p", "udp", "--dport", "123", "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append forward ntp skip rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_V4 -p tcp -j TPROXY --on-port 7893 --on-ip 0.0.0.0 --tproxy-mark 0x29a/0xffffffff
	err = m.ins.AppendUnique(tableMangle, chainIP4, "-p", "tcp", "-j", actionTProxy, "--on-port", m.cc.TProxyPort, "--tproxy-mark", conf.TproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_V4 -p udp -j TPROXY --on-port 7893 --on-ip 0.0.0.0 --tproxy-mark 0x29a/0xffffffff
	err = m.ins.AppendUnique(tableMangle, chainIP4, "-p", "udp", "-j", actionTProxy, "--on-port", m.cc.TProxyPort, "--tproxy-mark", conf.TproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append udp trpoxy rules: %v", err)
	}

	// iptables -t nat -A PREROUTING -d 198.18.0.0/16 -p icmp -j DNAT --to-destination 127.0.0.1
	err = m.ins.AppendUnique(tableNat, chainPreRouting, "-p", "icmp", "-d", m.cc.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to append icmp fake rules: %v", err)
	}

	return nil
}

func (m *tproxyMode) delForward() error {
	logrus.Debugf("[tproxy] delete forward iptables rules...")

	ok, err := m.ins.ChainExists(tableMangle, chainIP4)
	if err != nil {
		return fmt.Errorf("failed to check chain %s/%s: %s", tableMangle, chainIP4, err)
	}

	if ok {
		err = m.ins.DeleteIfExists(tableMangle, chainPreRouting, "-j", chainIP4)
		if err != nil {
			return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableMangle, chainPreRouting, chainIP4, err)
		}

		err = m.ins.ClearAndDeleteChain(tableMangle, chainIP4)
		if err != nil {
			return fmt.Errorf("failed to delete chain: %s/%s, error: %v", tableMangle, chainIP4, err)
		}
	}

	err = m.ins.DeleteIfExists(tableNat, chainPreRouting, "-p", "icmp", "-d", m.cc.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to delete icmp fake rules: %v", err)
	}

	return nil
}

func (m *tproxyMode) addForwardDNS() error {
	logrus.Debugf("[tproxy] add forward dns iptables rules...")

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

func (m *tproxyMode) delForwardDNS() error {
	logrus.Debugf("[tproxy] delete forward dns iptables rules...")

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

func (m *tproxyMode) addLocal() error {
	if !m.tpcc.LocalProxy {
		return nil
	}
	logrus.Debugf("[tproxy] add local iptables rules...")

	var err error

	// iptables -t mangle -N TP_CLASH_LOCAL_V4
	if err = createChain(m.ins, tableMangle, chainIP4Local); err != nil {
		return err
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -d 0.0.0.0/8 -j RETURN
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -d 127.0.0.0/8 -j RETURN
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -d 10.0.0.0/8 -j RETURN
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -d 172.16.0.0/12 -j RETURN
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -d 192.168.0.0/16 -j RETURN
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -d 169.254.0.0/16 -j RETURN
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -d 224.0.0.0/4 -j RETURN
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -d 240.0.0.0/4 -j RETURN
	if err = skipPrivateNetwork(m.ins, tableMangle, chainIP4Local); err != nil {
		return err
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -m owner --uid-owner m.tpcc.ClashUser -j RETURN
	err = m.ins.AppendUnique(tableMangle, chainIP4Local, "-m", "owner", "--uid-owner", m.tpcc.ClashUser, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway dns skip rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -m owner --gid-owner m.tpcc.DirectGroup -j RETURN
	err = m.ins.AppendUnique(tableMangle, chainIP4Local, "-m", "owner", "--gid-owner", m.tpcc.DirectGroup, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway group skip rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -m owner --gid-owner systemd-resolve -j RETURN
	if checkGroup(systemdResolveGroup) {
		err = m.ins.AppendUnique(tableMangle, chainIP4Local, "-m", "owner", "--gid-owner", systemdResolveGroup, "-j", actionReturn)
		if err != nil {
			return fmt.Errorf("failed to append gateway systemd-resolve skip rules: %v", err)
		}
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -p udp -m udp --dport 53 -j RETURN
	err = m.ins.AppendUnique(tableMangle, chainIP4Local, "-p", "udp", "--dport", "53", "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway dns skip rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -p udp -m udp --dport 123 -j RETURN
	err = m.ins.AppendUnique(tableMangle, chainIP4Local, "-p", "udp", "--dport", "123", "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway dns skip rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -p tcp -j MARK --set-xmark 0x29a/0xffffffff
	err = m.ins.AppendUnique(tableMangle, chainIP4Local, "-p", "tcp", "-j", actionMark, "--set-mark", m.tpcc.TproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -p udp -j MARK --set-xmark 0x29a/0xffffffff
	err = m.ins.AppendUnique(tableMangle, chainIP4Local, "-p", "udp", "-j", actionMark, "--set-mark", m.tpcc.TproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %v", err)
	}

	// iptables -t nat -A OUTPUT -d 198.18.0.0/16 -p icmp -j DNAT --to-destination 127.0.0.1
	err = m.ins.AppendUnique(tableNat, chainOutput, "-p", "icmp", "-d", m.cc.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to append icmp fake rules: %v", err)
	}

	return nil
}

func (m *tproxyMode) delLocal() error {
	if !m.tpcc.LocalProxy {
		return nil
	}
	logrus.Debugf("[tproxy] delete local iptables rules...")

	ok, err := m.ins.ChainExists(tableMangle, chainIP4Local)
	if err != nil {
		return fmt.Errorf("failed to check chain %s/%s: %s", tableMangle, chainIP4Local, err)
	}

	if ok {
		err = m.ins.DeleteIfExists(tableMangle, chainOutput, "-j", chainIP4Local)
		if err != nil {
			return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableMangle, chainOutput, chainIP4Local, err)
		}
		err = m.ins.ClearAndDeleteChain(tableMangle, chainIP4Local)
		if err != nil {
			return fmt.Errorf("failed to delete chain: %s/%s, error: %v", tableMangle, chainIP4Local, err)
		}
	}

	err = m.ins.DeleteIfExists(tableNat, chainOutput, "-p", "icmp", "-d", m.cc.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to delete icmp fake rules: %v", err)
	}

	return nil
}

func (m *tproxyMode) addLocalDNS() error {
	if !m.tpcc.LocalProxy {
		return nil
	}
	logrus.Debugf("[tproxy] add local dns iptables rules...")

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

func (m *tproxyMode) delLocalDNS() error {
	if !m.tpcc.LocalProxy {
		return nil
	}
	logrus.Debugf("[tproxy] delete local dns iptables rules...")

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

func (m *tproxyMode) addMisc() error {
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

func (m *tproxyMode) delMisc() error {
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

func (m *tproxyMode) apply() error {
	var err error

	logrus.Info("[route] add ip route rules...")
	bs, err := exec.Command("ip", "rule", "add", "fwmark", m.tpcc.TproxyMark, "lookup", m.tpcc.TproxyMark).CombinedOutput()
	if err != nil && !strings.Contains(string(bs), "File exists") {
		return fmt.Errorf("failed to create ip rule: %s, %v", bs, err)
	}

	logrus.Info("[route] add ip routes...")
	bs, err = exec.Command("ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", m.tpcc.TproxyMark).CombinedOutput()
	if err != nil && !strings.Contains(string(bs), "File exists") {
		return fmt.Errorf("failed to create ip route: %s, %v", bs, err)
	}

	logrus.Info("[iptables] apply all rules...")

	// iptables -t mangle -A PREROUTING -j TP_CLASH_V4
	err = m.ins.AppendUnique(tableMangle, chainPreRouting, "-j", chainIP4)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableMangle, chainPreRouting, chainIP4, err)
	}
	// iptables -t nat -A PREROUTING -j TP_CLASH_DNS_V4
	err = m.ins.AppendUnique(tableNat, chainPreRouting, "-j", chainIP4DNS)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableNat, chainPreRouting, chainIP4DNS, err)
	}

	if m.tpcc.LocalProxy {
		// iptables -t mangle -A OUTPUT -j TP_CLASH_LOCAL_V4
		err = m.ins.AppendUnique(tableMangle, chainOutput, "-j", chainIP4Local)
		if err != nil {
			return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableMangle, chainOutput, chainIP4Local, err)
		}
		// iptables -t nat -A OUTPUT -j TP_CLASH_DNS_LOCAL_V4
		err = m.ins.AppendUnique(tableNat, chainOutput, "-j", chainIP4DNSLocal)
		if err != nil {
			return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableNat, chainOutput, chainIP4DNSLocal, err)
		}
	}

	return nil
}

func (m *tproxyMode) clean() error {
	logrus.Info("[route] clean ip route rules...")
	bs, err := exec.Command("ip", "rule", "del", "fwmark", m.tpcc.TproxyMark, "table", m.tpcc.TproxyMark).CombinedOutput()
	if err != nil {
		logrus.Warnf("failed to clean route: %s, %v", strings.TrimSpace(string(bs)), err)
	}

	logrus.Info("[route] clean ip routes...")
	bs, err = exec.Command("ip", "route", "del", "local", "0.0.0.0/0", "dev", "lo", "table", m.tpcc.TproxyMark).CombinedOutput()
	if err != nil {
		logrus.Warnf("failed to clean route: %s, %v", strings.TrimSpace(string(bs)), err)
	}
	return nil
}

func (m *tproxyMode) EnableForward() error {
	return process(m.addForward, m.addForwardDNS, m.addLocal, m.addLocalDNS, m.addMisc, m.apply)
}

func (m *tproxyMode) DisableForward() error {
	return process(m.delForward, m.delForwardDNS, m.delLocal, m.delLocalDNS, m.delMisc, m.clean)
}

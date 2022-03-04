package main

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
)

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

func applyLocalNetwork(ins *iptables.IPTables, table, chain string) error {
	logrus.Debugf("[iptables] checking chain %s/%s rules...", table, chain)
	for _, cidr := range localCIDR {
		logrus.Debugf("[iptables] append local cidr %s rule to %s/%s...", cidr, table, chain)
		err := ins.AppendUnique(table, chain, "-d", cidr, "-j", actionReturn)
		if err != nil {
			return fmt.Errorf("failed to append local cidr rules: %v", err)
		}
	}
	return nil
}

func applyIPTables() error {
	/* Create *iptables.IPTables */

	ip4, err := newIPTables(iptables.ProtocolIPv4)
	if err != nil {
		return fmt.Errorf("faile to create ipv4 iptables instance: %v", err)
	}

	/* Forward Network Rules */

	// iptables -t mangle -N TP_CLASH_V4
	if err = createChain(ip4, tableMangle, chainIP4); err != nil {
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
	if err = applyLocalNetwork(ip4, tableMangle, chainIP4); err != nil {
		return err
	}

	// iptables -t mangle -A TP_CLASH_V4 -p udp -m udp --dport 53 -j RETURN
	err = ip4.AppendUnique(tableMangle, chainIP4, "-p", "udp", "--dport", "53", "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append forward dns skip rules: %v", err)
	}

	/* Gateway Network Rules */

	// iptables -t mangle -N TP_CLASH_LOCAL_V4
	if err = createChain(ip4, tableMangle, chainIP4Local); err != nil {
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
	if err = applyLocalNetwork(ip4, tableMangle, chainIP4Local); err != nil {
		return err
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -m owner --uid-owner tpclash -j RETURN
	err = ip4.AppendUnique(tableMangle, chainIP4Local, "-m", "owner", "--uid-owner", clashUser, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway dns skip rules: %v", err)
	}
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -m owner --gid-owner tpdirect -j RETURN
	err = ip4.AppendUnique(tableMangle, chainIP4Local, "-m", "owner", "--gid-owner", directGroup, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway user skip rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -p udp -m udp --dport 53 -j RETURN
	err = ip4.AppendUnique(tableMangle, chainIP4Local, "-p", "udp", "--dport", "53", "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway dns skip rules: %v", err)
	}

	/* TProxy Rules */

	logrus.Debug("[iptables] checking tproxy rules...")

	// iptables -t mangle -A TP_CLASH_V4 -p tcp -j TPROXY --on-port 7893 --on-ip 0.0.0.0 --tproxy-mark 0x29a/0xffffffff
	err = ip4.AppendUnique(tableMangle, chainIP4, "-p", "tcp", "-j", actionTProxy, "--on-port", conf.TProxyPort, "--tproxy-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %v", err)
	}
	// iptables -t mangle -A TP_CLASH_V4 -p udp -j TPROXY --on-port 7893 --on-ip 0.0.0.0 --tproxy-mark 0x29a/0xffffffff
	err = ip4.AppendUnique(tableMangle, chainIP4, "-p", "udp", "-j", actionTProxy, "--on-port", conf.TProxyPort, "--tproxy-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append udp trpoxy rules: %v", err)
	}

	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -p tcp -j MARK --set-xmark 0x29a/0xffffffff
	err = ip4.AppendUnique(tableMangle, chainIP4Local, "-p", "tcp", "-j", actionMark, "--set-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %v", err)
	}
	// iptables -t mangle -A TP_CLASH_LOCAL_V4 -p udp -j MARK --set-xmark 0x29a/0xffffffff
	err = ip4.AppendUnique(tableMangle, chainIP4Local, "-p", "udp", "-j", actionMark, "--set-mark", tproxyMark)
	if err != nil {
		return fmt.Errorf("failed to append tcp trpoxy rules: %v", err)
	}

	/* DNS Rules */

	// iptables -t nat -N TP_CLASH_DNS_V4
	if err = createChain(ip4, tableNat, chainIP4DNS); err != nil {
		return err
	}
	logrus.Debugf("[iptables] checking chain %s/%s rules...", tableNat, chainIP4DNS)
	// iptables -t nat -A TP_CLASH_DNS_V4 -p udp -m udp --dport 53 -j REDIRECT --to-ports 1053
	err = ip4.AppendUnique(tableNat, chainIP4DNS, "-p", "udp", "--dport", "53", "-j", actionRedirect, "--to", conf.DNSPort)
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %v", err)
	}

	// iptables -t nat -N TP_CLASH_DNS_LOCAL_V4
	if err = createChain(ip4, tableNat, chainIP4DNSLocal); err != nil {
		return err
	}
	logrus.Debugf("[iptables] checking chain %s/%s rules...", tableNat, chainIP4DNSLocal)
	// iptables -t nat -A TP_CLASH_DNS_LOCAL_V4 -m owner --uid-owner tpclash -j RETURN
	err = ip4.AppendUnique(tableNat, chainIP4DNSLocal, "-m", "owner", "--uid-owner", clashUser, "-j", actionReturn)
	// iptables -t nat -A TP_CLASH_DNS_LOCAL_V4 -m owner --gid-owner tpdirect -j RETURN
	err = ip4.AppendUnique(tableNat, chainIP4DNSLocal, "-m", "owner", "--gid-owner", directGroup, "-j", actionReturn)
	if err != nil {
		return fmt.Errorf("failed to append gateway group skip rules: %v", err)
	}
	// iptables -t nat -A TP_CLASH_DNS_LOCAL_V4 -p udp -m udp --dport 53 -j REDIRECT --to-ports 1053
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %v", err)
	}
	err = ip4.AppendUnique(tableNat, chainIP4DNSLocal, "-p", "udp", "--dport", "53", "-j", actionRedirect, "--to", conf.DNSPort)
	if err != nil {
		return fmt.Errorf("failed to append dns rules: %v", err)
	}

	/* Fix ICMP */

	logrus.Debug("[iptables] checking icmp fake rules...")

	// iptables -t nat -A PREROUTING -d 198.18.0.0/16 -p icmp -j DNAT --to-destination 127.0.0.1
	err = ip4.AppendUnique(tableNat, chainPreRouting, "-p", "icmp", "-d", conf.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to append icmp fake rules: %v", err)
	}

	// iptables -t nat -A OUTPUT -d 198.18.0.0/16 -p icmp -j DNAT --to-destination 127.0.0.1
	err = ip4.AppendUnique(tableNat, chainOutput, "-p", "icmp", "-d", conf.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to append icmp fake rules: %v", err)
	}

	/* Apply Rules */

	logrus.Info("[iptables] apply all rules...")
	err = ip4.DeleteIfExists(tableMangle, chainPreRouting, "-j", chainIP4)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableMangle, chainPreRouting, chainIP4, err)
	}
	err = ip4.DeleteIfExists(tableMangle, chainOutput, "-j", chainIP4Local)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableMangle, chainOutput, chainIP4Local, err)
	}
	err = ip4.DeleteIfExists(tableNat, chainPreRouting, "-j", chainIP4DNS)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableNat, chainPreRouting, chainIP4DNS, err)
	}
	err = ip4.DeleteIfExists(tableNat, chainOutput, "-j", chainIP4DNSLocal)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableNat, chainOutput, chainIP4DNSLocal, err)
	}

	// iptables -t mangle -A PREROUTING -j TP_CLASH_V4
	err = ip4.AppendUnique(tableMangle, chainPreRouting, "-j", chainIP4)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableMangle, chainPreRouting, chainIP4, err)
	}
	// iptables -t nat -A PREROUTING -j TP_CLASH_DNS_V4
	err = ip4.AppendUnique(tableNat, chainPreRouting, "-j", chainIP4DNS)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableNat, chainPreRouting, chainIP4DNS, err)
	}

	// iptables -t mangle -A OUTPUT -j TP_CLASH_LOCAL_V4
	err = ip4.AppendUnique(tableMangle, chainOutput, "-j", chainIP4Local)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableMangle, chainOutput, chainIP4Local, err)
	}
	// iptables -t nat -A OUTPUT -j TP_CLASH_DNS_LOCAL_V4
	err = ip4.AppendUnique(tableNat, chainOutput, "-j", chainIP4DNSLocal)
	if err != nil {
		return fmt.Errorf("failed to apply rules: %s/%s -> %s, error: %v", tableNat, chainOutput, chainIP4DNSLocal, err)
	}

	return nil
}

func cleanIPTables() {
	logrus.Info("[iptables] clean rules...")

	/* Create *iptables.IPTables */

	ip4, err := newIPTables(iptables.ProtocolIPv4)
	if err != nil {
		logrus.Errorf("faile to create ipv4 iptables instance: %v", err)
	}

	/* Clean iptables */

	ok, err := ip4.ChainExists(tableMangle, chainIP4)
	if err != nil {
		logrus.Errorf("failed to check chain %s/%s: %s", tableMangle, chainIP4, err)
	}

	if ok {
		logrus.Debugf("[iptables] clean %s/%s...", tableMangle, chainPreRouting)
		err = ip4.DeleteIfExists(tableMangle, chainPreRouting, "-j", chainIP4)
		if err != nil {
			logrus.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableMangle, chainPreRouting, chainIP4, err)
		}

		err = ip4.ClearAndDeleteChain(tableMangle, chainIP4)
		if err != nil {
			logrus.Errorf("failed to delete chain: %s/%s, error: %v", tableMangle, chainIP4, err)
		}
	}

	ok, err = ip4.ChainExists(tableMangle, chainIP4Local)
	if err != nil {
		logrus.Errorf("failed to check chain %s/%s: %s", tableMangle, chainIP4Local, err)
	}

	if ok {
		logrus.Debugf("[iptables] clean %s/%s...", tableMangle, chainOutput)
		err = ip4.DeleteIfExists(tableMangle, chainOutput, "-j", chainIP4Local)
		if err != nil {
			logrus.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableMangle, chainOutput, chainIP4Local, err)
		}
		err = ip4.ClearAndDeleteChain(tableMangle, chainIP4Local)
		if err != nil {
			logrus.Errorf("failed to delete chain: %s/%s, error: %v", tableMangle, chainIP4Local, err)
		}
	}

	ok, err = ip4.ChainExists(tableNat, chainIP4DNS)
	if err != nil {
		logrus.Errorf("failed to check chain %s/%s: %s", tableNat, chainIP4DNS, err)
	}
	if ok {
		logrus.Debug("[iptables] clean dns...")
		err = ip4.DeleteIfExists(tableNat, chainPreRouting, "-j", chainIP4DNS)
		if err != nil {
			logrus.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableNat, chainPreRouting, chainIP4DNS, err)
		}
		err = ip4.ClearAndDeleteChain(tableNat, chainIP4DNS)
		if err != nil {
			logrus.Errorf("failed to delete chain: %s/%s, error: %v", tableNat, chainIP4DNS, err)
		}
	}

	ok, err = ip4.ChainExists(tableNat, chainIP4DNSLocal)
	if err != nil {
		logrus.Errorf("failed to check chain %s/%s: %s", tableNat, chainIP4DNSLocal, err)
	}
	if ok {
		logrus.Debug("[iptables] clean local dns...")
		err = ip4.DeleteIfExists(tableNat, chainOutput, "-j", chainIP4DNSLocal)
		if err != nil {
			logrus.Errorf("failed to delete rules: %s/%s -> %s, error: %v", tableNat, chainOutput, chainIP4DNSLocal, err)
		}
		err = ip4.ClearAndDeleteChain(tableNat, chainIP4DNSLocal)
		if err != nil {
			logrus.Errorf("failed to delete chain: %s/%s, error: %v", tableNat, chainIP4DNS, err)
		}
	}

	logrus.Debug("[iptables] clean icmp fake...")
	err = ip4.DeleteIfExists(tableNat, chainPreRouting, "-p", "icmp", "-d", conf.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		logrus.Errorf("failed to delete icmp fake rules: %v", err)
	}

	err = ip4.DeleteIfExists(tableNat, chainOutput, "-p", "icmp", "-d", conf.FakeIPRange, "-j", actionDNat, "--to-destination", "127.0.0.1")
	if err != nil {
		logrus.Errorf("failed to delete icmp fake rules: %v", err)
	}
}

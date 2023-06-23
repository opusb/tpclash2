package main

import (
	"fmt"
	"net"

	"github.com/google/nftables/expr"

	"github.com/google/nftables"

	"github.com/lorenzosaino/go-sysctl"
	"github.com/sirupsen/logrus"
)

func Sysctl() {
	logrus.Info("[helper/sysctl] enable net.ipv4.ip_forward...")
	if err := sysctl.Set("net.ipv4.ip_forward", "1"); err != nil {
		logrus.Fatalf("[helper] failed to set net.ipv4.ip_forward: %v", err)
	}

	logrus.Info("[helper/sysctl] enable net.ipv4.conf.all.route_localnet...")
	if err := sysctl.Set("net.ipv4.conf.all.route_localnet", "1"); err != nil {
		logrus.Fatalf("[helper/sysctl] failed to set net.ipv4.conf.all.route_localnet: %v", err)
	}
}

func IfName() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		logrus.Errorf("[helper/ifname] failed to list network interfaces: %v", err)
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				logrus.Errorf("[helper/ifname] failed to get addrs: %v", err)
				return ""
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return iface.Name
					}
				}
			}
		}
	}

	logrus.Error("[helper/ifname] failed to get main interface")
	return ""
}

func EnableDockerCompatible() error {
	nft, err := nftables.New()
	if err != nil {
		return fmt.Errorf("[helper/nftables] failed connect to nftables: %v", err)
	}

	cs, err := nft.ListChainsOfTableFamily(nftables.TableFamilyIPv4)
	if err != nil {
		return fmt.Errorf("[helper/nftables] failed to list nftables chain: %w", err)
	}
	for _, chain := range cs {
		if chain.Name == ChainDockerUser {
			nft.InsertRule(&nftables.Rule{
				Table: chain.Table,
				Chain: chain,
				Exprs: []expr.Any{&expr.Verdict{
					Kind: expr.VerdictAccept,
				}},
			})
			if err = nft.Flush(); err != nil {
				return fmt.Errorf("[helper/nftables] failed to flush nftables: %v", err)
			}
			return nil
		}
	}
	return nil
}

func DisableDockerCompatible() error {
	nft, err := nftables.New()
	if err != nil {
		return fmt.Errorf("[helper/nftables] failed connect to nftables: %v", err)
	}

	cs, err := nft.ListChainsOfTableFamily(nftables.TableFamilyIPv4)
	if err != nil {
		return fmt.Errorf("[helper/nftables] failed to list nftables chain: %w", err)
	}
	for _, chain := range cs {
		if chain.Name == ChainDockerUser {
			rs, err := nft.GetRules(chain.Table, chain)
			if err != nil {
				return fmt.Errorf("[helper/nftables] failed to get nftables rules: %w", err)
			}
			for _, rule := range rs {
				if len(rule.Exprs) == 1 {
					v, ok := rule.Exprs[0].(*expr.Verdict)
					if ok && v.Kind == expr.VerdictAccept {
						if err = nft.DelRule(rule); err != nil {
							return fmt.Errorf("[helper/nftables] failed to delete nftables rules: %w", err)
						}
					}
				}
			}
			if err = nft.Flush(); err != nil {
				return fmt.Errorf("[helper/nftables] failed to flush nftables: %v", err)
			}
			return nil
		}
	}
	return nil
}

package main

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/google/nftables/expr"

	"github.com/google/nftables"
)

type tunMode struct {
	nft  *nftables.Conn
	tpcc *TPClashConf
	cc   *ClashConf
}

func (m *tunMode) addMisc() error {
	cs, err := m.nft.ListChainsOfTableFamily(nftables.TableFamilyIPv4)
	if err != nil {
		logrus.Errorf("[nftables] failed to list nftables chain: %v", err)
		return nil
	}
	for _, chain := range cs {
		if chain.Name == chainDockerUser {
			m.nft.InsertRule(&nftables.Rule{
				Table: chain.Table,
				Chain: chain,
				Exprs: []expr.Any{&expr.Verdict{
					Kind: expr.VerdictAccept,
				}},
			})
			return m.nft.Flush()
		}
	}
	return nil
}

func (m *tunMode) delMisc() error {
	cs, err := m.nft.ListChainsOfTableFamily(nftables.TableFamilyIPv4)
	if err != nil {
		logrus.Errorf("[nftables] failed to list nftables chain: %v", err)
		return nil
	}
	for _, chain := range cs {
		if chain.Name == chainDockerUser {
			rs, err := m.nft.GetRules(chain.Table, chain)
			if err != nil {
				return fmt.Errorf("failed to get nftables rules: %v", err)
			}
			for _, rule := range rs {
				if len(rule.Exprs) == 1 {
					v, ok := rule.Exprs[0].(*expr.Verdict)
					if ok && v.Kind == expr.VerdictAccept {
						if err = m.nft.DelRule(rule); err != nil {
							return fmt.Errorf("failed to delete nftables rules: %v", err)
						}
					}
				}
			}

			return m.nft.Flush()
		}
	}
	return nil
}

func (m *tunMode) EnableProxy() error {
	return process(m.addMisc)
}

func (m *tunMode) DisableProxy() error {
	return process(m.delMisc)
}

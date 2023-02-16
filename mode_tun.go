package main

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"
)

type tunMode struct {
	ins  *iptables.IPTables
	tpcc *TPClashConf
	cc   *ClashConf
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
	// iptables -t filter -D DOCKER-USER -j ACCEPT
	err := m.ins.DeleteIfExists(tableFilter, chainDockerUser, "-j", actionAccept)
	if err != nil {
		return fmt.Errorf("failed to delete docker rules: %v", err)
	}
	return nil
}

func (m *tunMode) EnableProxy() error {
	return process(m.addMisc)
}

func (m *tunMode) DisableProxy() error {
	return process(m.delMisc)
}

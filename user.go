package main

import (
	"os/exec"
	"os/user"

	"github.com/sirupsen/logrus"
)

func ensureUserAndGroup() {
	if !checkUser(conf.ClashUser) {
		bs, err := exec.Command("useradd", "-M", "-s", "/bin/false", conf.ClashUser).CombinedOutput()
		if err != nil {
			logrus.Fatalf("failed to create tpclash user: %s, %v", bs, err)
		}
	}
	if !checkGroup(conf.DirectGroup) {
		bs, err := exec.Command("groupadd", conf.DirectGroup).CombinedOutput()
		if err != nil {
			logrus.Fatalf("failed to create tpdirect group: %s, %v", bs, err)
		}
	}
}

func checkUser(u string) bool {
	ou, err := user.Lookup(u)
	if err != nil {
		return false
	}
	return ou != nil
}

func checkGroup(g string) bool {
	og, err := user.LookupGroup(g)
	if err != nil {
		return false
	}
	return og != nil
}

func chownR(p string) error {
	bs, err := exec.Command("chown", "-R", conf.ClashUser+":"+conf.ClashUser, p).CombinedOutput()
	if err != nil {
		logrus.Fatalf("failed to change dir owner: %s, %v", bs, err)
	}
	return nil
}

func chmod(p string) error {
	bs, err := exec.Command("chmod", "+x", p).CombinedOutput()
	if err != nil {
		logrus.Fatalf("failed to change file permission: %s, %v", bs, err)
	}
	return nil
}

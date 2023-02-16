package main

import (
	"os/exec"
	"os/user"
	"strconv"

	"github.com/sirupsen/logrus"
)

func ensureUser() {
	if !checkUser(conf.ClashUser) {
		bs, err := exec.Command("useradd", "-M", "-s", "/bin/false", conf.ClashUser).CombinedOutput()
		if err != nil {
			logrus.Fatalf("failed to create tpclash user: %s, %v", bs, err)
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

func getUserIDs(u string) (uint32, uint32) {
	us, _ := user.Lookup(u)

	uid, _ := strconv.Atoi(us.Uid)
	gid, _ := strconv.Atoi(us.Gid)
	return uint32(uid), uint32(gid)
}

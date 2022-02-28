package main

import (
	"os/exec"
	"os/user"

	"github.com/sirupsen/logrus"
)

func createUser() {
	if !checkUser() {
		bs, err := exec.Command("useradd", "-M", "-s", "/bin/false", clashUser).CombinedOutput()
		if err != nil {
			logrus.Fatalf("failed to create tpclash user: %s, %v", bs, err)
		}
	}
}

func checkUser() bool {
	u, err := user.Lookup(clashUser)
	if err != nil {
		return false
	}
	return u != nil
}

func chownR(p string) error {
	bs, err := exec.Command("chown", "-R", clashUser, clashUser, p).CombinedOutput()
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

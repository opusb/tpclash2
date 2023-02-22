package main

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

//go:embed static
var static embed.FS

func mkHomeDir() {
	info, err := os.Stat(conf.ClashHome)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(conf.ClashHome, 0755)
			if err != nil {
				logrus.Fatalf("failed to create clash home dir: %v", err)
			}
		} else {
			logrus.Fatalf("failed to open clash home dir: %v", err)
		}
	} else if !info.IsDir() {
		logrus.Fatalf("clash home path is not a dir")
	}
}

func copyFiles() {
	if conf.DisableExtract {
		logrus.Warn("[static] skip copy static files...")
		return
	}

	logrus.Info("[static] copy static files...")

	dirEntries, err := static.ReadDir("static")
	if err != nil {
		logrus.Fatal(err)
	}
	err = extract(static, dirEntries, "static", conf.ClashHome)
	if err != nil {
		logrus.Fatal(err)
	}

	bs, err := exec.Command("chmod", "+x", filepath.Join(conf.ClashHome, "xclash")).CombinedOutput()
	if err != nil {
		logrus.Fatalf("[static] failed to change file permission: %s, %v", bs, err)
	}
}

func extract(efs embed.FS, dirEntries []fs.DirEntry, origin, target string) error {
	for _, dirEntry := range dirEntries {
		info, err := dirEntry.Info()
		if err != nil {
			return err
		}
		perm := info.Mode().Perm()

		if dirEntry.IsDir() {
			logrus.Debugf("[static] extract -> %s %s", filepath.Join(target, dirEntry.Name()), perm.String())
			err := os.MkdirAll(filepath.Join(target, dirEntry.Name()), perm)
			if err != nil {
				return err
			}
			entries, err := efs.ReadDir(filepath.Join(origin, dirEntry.Name()))
			if err != nil {
				return err
			}
			err = extract(efs, entries, filepath.Join(origin, dirEntry.Name()), filepath.Join(target, dirEntry.Name()))
			if err != nil {
				return err
			}
		} else {
			sf, err := static.Open(filepath.Join(origin, dirEntry.Name()))
			if err != nil {
				return err
			}
			defer func() { _ = sf.Close() }()

			logrus.Debugf("[static] extract -> %s %s", filepath.Join(target, dirEntry.Name()), perm.String())
			df, err := os.OpenFile(filepath.Join(target, dirEntry.Name()), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
			if err != nil {
				return err
			}
			defer func() { _ = df.Close() }()

			_, err = io.Copy(df, sf)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureClashFiles() {
	mkHomeDir()
	copyFiles()
}

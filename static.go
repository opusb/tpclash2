package main

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

//go:embed static
var static embed.FS

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

func ExtractFiles() {
	logrus.Info("[static] creating storage dir...")
	info, err := os.Stat(conf.ClashHome)
	if err == nil {
		if !info.IsDir() {
			logrus.Fatalf("[static] clash home path is not a dir")
		}

		if !conf.ForceExtract {
			logrus.Infof("[static] storage dir %s already exist, skip extract...", conf.ClashHome)
			return
		}
	} else {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(conf.ClashHome, 0755); err != nil {
				logrus.Fatalf("[static] failed to create storage dir: %v", err)
			}
		} else {
			logrus.Fatalf("[static] failed to read storage dir: %v", err)
		}
	}

	logrus.Info("[static] copy static files...")
	dirEntries, err := static.ReadDir("static")
	if err != nil {
		logrus.Fatalf("[static] failed to read embed dir: %v", err)
	}

	err = extract(static, dirEntries, "static", conf.ClashHome)
	if err != nil {
		logrus.Fatalf("[static] failed to extract embed files: %v", err)
	}

	err = os.Chmod(filepath.Join(conf.ClashHome, InternalClashBinName), 0755)
	if err != nil {
		logrus.Fatalf("[static] failed to update internal clash bin mode: %v", err)
	}
}

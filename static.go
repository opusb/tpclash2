package main

import (
	"embed"
	"github.com/sirupsen/logrus"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed static
var static embed.FS

func mkHomeDir() {
	info, err := os.Stat(clashHome)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(clashHome, 0755)
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
	logrus.Info("[static] copy static files...")
	info, err := os.Stat(clashHome)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(clashHome, 0755)
			if err != nil {
				logrus.Fatalf("failed to create clash home dir: %v", err)
			}
		} else {
			logrus.Fatalf("failed to open clash home dir: %v", err)
		}
	} else if !info.IsDir() {
		logrus.Fatalf("clash home path is not a dir")
	}

	dirEntries, err := static.ReadDir("static")
	if err != nil {
		logrus.Fatal(err)
	}
	err = extract(static, dirEntries, "static", clashHome)
	if err != nil {
		logrus.Fatal(err)
	}
}

func extract(efs embed.FS, dirEntries []fs.DirEntry, origin, target string) error {
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			logrus.Debugf("[static] extract -> %s", filepath.Join(target, dirEntry.Name()))
			err := os.MkdirAll(filepath.Join(target, dirEntry.Name()), dirEntry.Type())
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

			logrus.Debugf("[static] extract -> %s", filepath.Join(target, dirEntry.Name()))
			df, err := os.OpenFile(filepath.Join(target, dirEntry.Name()), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, dirEntry.Type())
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

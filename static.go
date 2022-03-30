package main

import (
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	err = extract(static, dirEntries, "static", conf.ClashHome, conf.MMDB, conf.ClashURL == "")
	if err != nil {
		logrus.Fatal(err)
	}

	if conf.ClashURL != "" {
		logrus.Info("[static] downloading clash...")
		if err = downloadClash(conf.ClashURL, filepath.Join(conf.ClashHome, "xclash")); err != nil {
			logrus.Fatal(err)
		}
	}

	err = chmod(filepath.Join(conf.ClashHome, "xclash"))
	if err != nil {
		logrus.Fatal(err)
	}

	err = chownR(conf.ClashHome)
	if err != nil {
		logrus.Fatal(err)
	}
}

func extract(efs embed.FS, dirEntries []fs.DirEntry, origin, target string, mmdb, xclash bool) error {
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
			err = extract(efs, entries, filepath.Join(origin, dirEntry.Name()), filepath.Join(target, dirEntry.Name()), mmdb, xclash)
			if err != nil {
				return err
			}
		} else {
			if !mmdb && strings.Contains(dirEntry.Name(), "Country.mmdb") {
				logrus.Warn("[static] skip extract mmdb...")
				continue
			}
			if !xclash && strings.Contains(dirEntry.Name(), "xclash") {
				logrus.Warn("[static] skip extract xclash...")
				continue
			}
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

func downloadClash(u, target string) error {
	_, err := os.Stat(target)
	if err == nil {
		logrus.Warn("[static] xclash already exist, skip download...")
		return nil
	} else {
		if !os.IsNotExist(err) {
			return fmt.Errorf("[static] failed to check xclash status: %s", err)
		}
	}
	resp, err := http.Get(u)
	if err != nil {
		return fmt.Errorf("[static] failed to download clash: %s", err)
	}
	defer func() { _ = resp.Body.Close() }()

	r, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("[static] failed to create gzip reader: %s", err)
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("[static] failed to create xclash: %s", err)
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, r)
	if err != nil {
		return fmt.Errorf("[static] failed to create xclash: %s", err)
	}

	return nil
}

func ensureClashFiles() {
	mkHomeDir()
	copyFiles()
}

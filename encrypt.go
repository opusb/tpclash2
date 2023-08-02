package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var encCmd = &cobra.Command{
	Use:   "enc FILENAME",
	Short: "Encrypt config file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			_ = cmd.Help()
			return
		}
		if conf.ConfigEncPassword == "" {
			logrus.Fatalf("[enc] configuration file encryption password cannot be empty")
		}

		plaintext, err := os.ReadFile(args[0])
		if err != nil {
			logrus.Fatalf("[enc] failed to read config file: %v", err)
		}

		ciphertext := Encrypt(plaintext, conf.ConfigEncPassword)
		if err = os.WriteFile(args[0]+".enc", ciphertext, 0644); err != nil {
			logrus.Fatalf("[enc] failed to write encrypted config file: %v", err)
		}

		logrus.Infof("[enc] encrypted file storage location %s", args[0]+".enc")
	},
}

var decCmd = &cobra.Command{
	Use:   "dec FILENAME",
	Short: "Decrypt config file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			_ = cmd.Help()
			return
		}
		if conf.ConfigEncPassword == "" {
			logrus.Fatalf("[dec] configuration file encryption password cannot be empty")
		}

		ciphertext, err := os.ReadFile(args[0])
		if err != nil {
			logrus.Fatalf("[dec] failed to read encrypted config file: %v", err)
		}

		plaintext, err := Decrypt(ciphertext, conf.ConfigEncPassword)
		if err != nil {
			logrus.Fatalf("[dec] failed to decrypt config file: %v", err)
		}

		if err = os.WriteFile(strings.TrimSuffix(args[0], ".enc"), plaintext, 0644); err != nil {
			logrus.Fatalf("[dec] failde to write config file: %v", err)
		}

		logrus.Infof("[enc] decrypted file storage location %s", strings.TrimSuffix(args[0], ".enc"))
	},
}

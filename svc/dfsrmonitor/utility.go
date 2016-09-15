// +build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/adsi.v0"
)

func exePath() (string, error) {
	prog := os.Args[0]
	path, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}

	err = checkPath(path)
	if err == nil {
		return path, nil
	}

	if filepath.Ext(path) == "" {
		path += ".exe"
		err = checkPath(path)
		if err == nil {
			return path, nil
		}
	}

	return "", err
}

func checkPath(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Mode().IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}

func dnc(client *adsi.Client) (dnc string, err error) {
	rootDSE, err := client.Open("LDAP://RootDSE")
	if err != nil {
		return
	}
	defer rootDSE.Close()

	return rootDSE.AttrString("rootDomainNamingContext")
}

// +build windows

package main

import (
	adsi "gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/core"
	"gopkg.in/dfsr.v0/dfsrconfig"
)

func resolve(domain string) (dom string, data core.Domain, err error) {
	client, err := adsi.NewClient()
	if err != nil {
		return
	}
	defer client.Close()

	if domain == "" {
		domain, err = dnc(client)
		if err != nil {
			return
		}
	}
	dom = domain

	data, err = dfsrconfig.Domain(client, domain)

	return
}

func dnc(client *adsi.Client) (dnc string, err error) {
	rootDSE, err := client.Open("LDAP://RootDSE")
	if err != nil {
		return
	}
	defer rootDSE.Close()

	return rootDSE.AttrString("rootDomainNamingContext")
}

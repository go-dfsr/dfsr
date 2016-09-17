package config

import "gopkg.in/adsi.v0"

func dnc(client *adsi.Client) (dnc string, err error) {
	rootDSE, err := client.Open("LDAP://RootDSE")
	if err != nil {
		return
	}
	defer rootDSE.Close()

	return rootDSE.AttrString("rootDomainNamingContext")
}

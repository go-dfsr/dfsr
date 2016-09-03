package config

import (
	"strings"

	"gopkg.in/adsi.v0"
)

func domainPath(dn string) string {
	return "LDAP://" + dn
}

func domainDN(domain string) string {
	domain = strings.ToLower(domain)
	if strings.Index(strings.ToLower(domain), "dc=") == 0 {
		return domain
	}
	return makeDN("dc", strings.Split(domain, ".")...)
}

func makeDN(attribute string, components ...string) string {
	if attribute != "" {
		for i := 0; i < len(components); i++ {
			components[i] = attribute + "=" + components[i]
		}
	}
	return strings.Join(components, ",")
}

func combineDN(components ...string) string {
	return strings.Join(components, ",")
}

func dnc(client *adsi.Client) (dnc string, err error) {
	rootDSE, err := client.Open("LDAP://RootDSE")
	if err != nil {
		return
	}
	defer rootDSE.Close()

	return rootDSE.AttrString("rootDomainNamingContext")
}

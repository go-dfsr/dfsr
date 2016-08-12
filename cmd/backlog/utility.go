package main

import "gopkg.in/adsi.v0"

func dnc(client *adsi.Client) (dnc string, err error) {
	rootDSE, err := client.Open("LDAP://RootDSE")
	if err != nil {
		return
	}
	defer rootDSE.Close()

	return rootDSE.AttrString("rootDomainNamingContext")
}

func isMatch(s string, rs regexSlice, emptyValue bool) bool {
	if len(rs) == 0 {
		return emptyValue
	}

	for _, re := range rs {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

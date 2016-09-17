package globalsettings

import "strings"

func ldap(dn string) string {
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

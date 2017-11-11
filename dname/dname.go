package dname

import "strings"

// URL converts the given distinguished name into URL-form using the LDAP
// protocol.
func URL(dn string) string {
	return "LDAP://" + dn
}

// Domain converts the given domain DNS address into a distinguished name.
func Domain(domain string) string {
	domain = strings.ToLower(domain)
	if strings.HasPrefix(strings.ToLower(domain), "dc=") {
		return domain
	}
	return Make("dc", strings.Split(domain, ".")...)
}

// Make creates a new distinguished name from the given components. Each
// component will be of the given attribute type.
func Make(attribute string, components ...string) string {
	if attribute != "" {
		for i := 0; i < len(components); i++ {
			components[i] = attribute + "=" + components[i]
		}
	}
	return strings.Join(components, ",")
}

// Combine combines the given distinguished name components.
func Combine(components ...string) string {
	return strings.Join(components, ",")
}

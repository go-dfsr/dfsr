package globalsettings

import "time"

const groupQueryDelay = 25 * time.Millisecond // Group query delay to avoid rate-limiting by LDAP servers

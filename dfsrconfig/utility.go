package dfsrconfig

import (
	"github.com/google/uuid"
	"gopkg.in/adsi.v0"
)

func dnc(client *adsi.Client) (dnc string, err error) {
	rootDSE, err := client.Open("LDAP://RootDSE")
	if err != nil {
		return
	}
	defer rootDSE.Close()

	return rootDSE.AttrString("rootDomainNamingContext")
}

func littleEndianToBigEndian(id *uuid.UUID) {
	id[0], id[1], id[2], id[3] = id[3], id[2], id[1], id[0]
	id[4], id[5] = id[5], id[4]
	id[6], id[7] = id[7], id[6]
}

package main

import (
	"flag"
	"fmt"
	"log"

	"gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/config"
)

func main() {
	flag.Parse()

	client, err := adsi.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	var domain = flag.Arg(0)

	if domain == "" {
		dnc, dncErr := rootDNC(client)
		if dncErr != nil {
			log.Fatal(dncErr)
		}
		domain = dnc
	}

	d, err := config.Domain(client, domain)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("      Domain: %-51s ID: %v DN: %-30s Duration: %v\n", d.Description, d.ID, d.DN, d.ConfigDuration)
	for i := 0; i < len(d.Groups); i++ {
		group := &d.Groups[i]
		fmt.Printf("[%3d]   Group: %-50s ID: %v Duration: %v\n", i, group.Name, group.ID, group.ConfigDuration)
		for f := 0; f < len(group.Folders); f++ {
			folder := &group.Folders[f]
			fmt.Printf("          Folder: %-47s ID: %v\n", folder.Name, folder.ID)
		}
		for m := 0; m < len(group.Members); m++ {
			member := &group.Members[m]
			fmt.Printf("          Member: %-47s ID: %v Computer: %s\n", member.Name, member.ID, member.Computer.Host)
			for c := 0; c < len(member.Connections); c++ {
				conn := &member.Connections[c]
				fmt.Printf("            Connection: %-41s ID: %v Computer: %s\n", conn.Name, conn.ID, conn.Computer.Host)
			}
		}
	}
	fmt.Printf("Duration: %v\n", d.ConfigDuration)
}

func rootDNC(client *adsi.Client) (string, error) {
	rootDSE, err := client.Open("LDAP://RootDSE")
	if err != nil {
		return "", err
	}
	defer rootDSE.Close()

	return rootDSE.AttrString("rootDomainNamingContext")
}

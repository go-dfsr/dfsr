package dfsrconfig

import (
	"strings"

	adsi "gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/dfsr"
)

func (c *Client) folders(content *adsi.Container) (folders []dfsr.Folder, err error) {
	iter, err := content.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for f, err := iter.Next(); err == nil; f, err = iter.Next() {
		defer f.Close()

		folder, err := c.folder(f)
		if err != nil {
			return nil, err
		}

		folders = append(folders, folder)
	}

	return
}

func (c *Client) folder(f *adsi.Object) (folder dfsr.Folder, err error) {
	folder.Name, err = f.Name()
	if err != nil {
		return
	}
	folder.Name = strings.TrimPrefix(folder.Name, "CN=")

	folder.ID, err = f.GUID()
	if err != nil {
		return
	}

	return
}

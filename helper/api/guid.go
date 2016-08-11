package api

import (
	ole "github.com/go-ole/go-ole"
)

var (
	// CLSID_DFSRHelper is the component object model identifier of the DfsrHelper
	// class.
	CLSID_DFSRHelper = ole.NewGUID("{3B35075C-01ED-45BC-9999-DC2BBDEAC171}")

	// IID_IServerHealthReport is the component object model identifier of the
	// IServerHealthReport interface.
	IID_IServerHealthReport = ole.NewGUID("{E65E8028-83E8-491B-9AF7-AAF6BD51A0CE}")

	// IID_IServerHealthReport2 is the component object model identifier of the
	// IServerHealthReport2 interface.
	IID_IServerHealthReport2 = ole.NewGUID("{20D15747-6C48-4254-A358-65039FD8C63C}")
)

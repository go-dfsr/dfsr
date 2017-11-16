package api

import "github.com/google/uuid"

var (
	// CLSID_DFSRHelper is the component object model identifier of the DfsrHelper
	// class.
	//
	// CLSID_DFSRHelper
	// {3B35075C-01ED-45BC-9999-DC2BBDEAC171}
	CLSID_DFSRHelper = uuid.UUID{0x3B, 0x35, 0x07, 0x5C, 0x01, 0xED, 0x45, 0xBC, 0x99, 0x99, 0xDC, 0x2B, 0xBD, 0xEA, 0xC1, 0x71}

	// IID_IServerHealthReport is the component object model identifier of the
	// IServerHealthReport interface.
	//
	// IID_IServerHealthReport
	// {E65E8028-83E8-491B-9AF7-AAF6BD51A0CE}
	IID_IServerHealthReport = uuid.UUID{0xE6, 0x5E, 0x80, 0x28, 0x83, 0xE8, 0x49, 0x1B, 0x9A, 0xF7, 0xAA, 0xF6, 0xBD, 0x51, 0xA0, 0xCE}

	// IID_IServerHealthReport2 is the component object model identifier of the
	// IServerHealthReport2 interface.
	//
	// IID_IServerHealthReport2
	// {20D15747-6C48-4254-A358-65039FD8C63C}
	IID_IServerHealthReport2 = uuid.UUID{0x20, 0xD1, 0x57, 0x47, 0x6C, 0x48, 0x42, 0x54, 0xA3, 0x58, 0x65, 0x03, 0x9F, 0xD8, 0xC6, 0x3C}
)

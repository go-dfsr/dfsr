package manifest

import (
	"encoding/xml"
	"time"

	"github.com/google/uuid"
)

// Resource is a DFSR manifest resource entry.
type Resource struct {
	resource
	PartnerGUID uuid.UUID `xml:"PartnerGuid"`
	PartnerHost string    `xml:"PartnerHost,omitempty"`
	PartnerDN   string    `xml:"PartnerDN,omitempty"`
	Type        string    `xml:"Type"`
	Time        time.Time `xml:"Time"`
}

type resource struct {
	Path       string `xml:"Path"`
	UID        string `xml:"Uid"`
	GVSN       string `xml:"Gvsn"`
	Attributes string `xml:"Attributes"`
	NewName    string `xml:"NewName"`
	Files      int    `xml:"Files"`
	Size       int64  `xml:"Size"`
}

type rawResource struct {
	*resource
	PartnerGUID string `xml:"PartnerGuid"`
	Type        Type   `xml:"Type"`
	Time        Time   `xml:"Time"`
}

// UnmarshalXML decodes DFSR manifest time values.
func (r *Resource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	raw := rawResource{resource: &r.resource}
	d.DecodeElement(&raw, &start)
	r.Time = time.Time(raw.Time)
	r.Type = string(raw.Type)
	if raw.PartnerGUID != "" {
		r.PartnerGUID, err = uuid.Parse(raw.PartnerGUID)
	}
	return
}

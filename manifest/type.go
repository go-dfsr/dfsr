package manifest

import (
	"encoding/xml"
)

// Type represents a manifest resource entry type.
type Type string

type rawType struct {
	Element element `xml:",any"`
}

type element struct {
	XMLName xml.Name
}

// UnmarshalXML decodes DFSR manifest type values.
func (t *Type) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	var raw rawType
	d.DecodeElement(&raw, &start)
	*t = Type(raw.Element.XMLName.Local)
	return
}

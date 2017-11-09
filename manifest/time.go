package manifest

import (
	"encoding/xml"
	"time"
)

// TimeFormat describes the DFSR manifest time format.
const TimeFormat = "MST 2006:1:2-15:4:5"

// Time is a time.Time that can be used by an XML decoder to deserialize
// manifest time values.
type Time time.Time

// UnmarshalXML decodes DFSR manifest time values.
func (t *Time) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	var v string
	d.DecodeElement(&v, &start)
	parsed, err := time.Parse(TimeFormat, v)
	*t = Time(parsed)
	return
}

// UnmarshalXMLAttr decodes DFSR manifest time value attributes.
func (t *Time) UnmarshalXMLAttr(attr xml.Attr) (err error) {
	parsed, err := time.Parse(TimeFormat, attr.Value)
	*t = Time(parsed)
	return
}

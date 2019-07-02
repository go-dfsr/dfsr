package manifest

import (
	"encoding/xml"
	"fmt"
	"io"
)

const resourceElement = "Resource"

// Decoder reads and decodes DFSR conflict and deleted manifest entries from an
// input stream.
type Decoder struct {
	stream *xml.Decoder
	count  int64
}

// NewDecoder returns a DFSR conflicted and deleted manifest decoder that reads
// from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{stream: xml.NewDecoder(r)}
}

// Read returns the next resource record from the manifest data stream.
func (d *Decoder) Read() (resource Resource, err error) {
	for {
		var token interface{}
		token, err = d.stream.Token()
		if err != nil {
			return
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == resourceElement {
				d.count++
				if err = d.stream.DecodeElement(&resource, &se); err != nil {
					err = fmt.Errorf("manifest.Decoder: element %d: %v", d.count, err)
				}
				return
			}
		default:
		}
	}
}

// Count returns the number of resource records contained in the manifest.
func (d *Decoder) Count() (total int, err error) {
	for {
		var token interface{}
		token, err = d.stream.Token()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == resourceElement {
				total++
			}
		default:
		}
	}
}

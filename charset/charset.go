package charset

import (
	"fmt"
)

type Charset = string

const (
	ShiftJis Charset = "shift-jis"
	UTF8     Charset = "utf-8"
)

type Decoder struct {
	m map[string]Charset
}

var Charsets = []Charset{
	UTF8,
	ShiftJis,
}

// NewDecoder returns a Decoder
//
// This Map is handling Character set translation utility.
func NewDecoder() *Decoder {
	return &Decoder{
		m: make(map[string]Charset),
	}
}

// Register the desired character code conversion process according to the sender's address.
func (c *Decoder) Register(addr, charset string) error {
	for i := range Charsets {
		if charset == Charsets[i] {
			c.m[addr] = Charsets[i]
			return nil
		}
	}
	return fmt.Errorf("charset is missing. %q", charset)
}

// Decode binary to string.
func (c *Decoder) Decode(addr string, b []byte) (string, error) {
	var val string
	var err error
	switch c.m[addr] {
	case ShiftJis:
		val, err = transformShiftJIS(b)
		if err != nil {
			return "", err
		}
	case UTF8:
		fallthrough
	default:
		val = string(b)
	}
	return val, nil
}

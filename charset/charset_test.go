package charset

import (
	"testing"
)

func TestAdd(t *testing.T) {
	t.Run("valid charset name", func(t *testing.T) {
		c := NewDecoder()
		err := c.Register("192.0.2.1", "utf-8")
		if err != nil {
			t.Errorf("error should be nil but: %+v", err)
		}
	})

	t.Run("invalid charset name", func(t *testing.T) {
		c := NewDecoder()
		actual := c.Register("192.0.2.1", "invalid-charset-name")
		if actual == nil {
			t.Errorf("should return error")
		}
		expected := "charset is missing. \"invalid-charset-name\""
		if actual.Error() != expected {
			t.Errorf("error should be %+v but: %+v", expected, actual.Error())
		}
	})
}

func TestDecoder(t *testing.T) {
	d := NewDecoder()
	d.Register("192.0.2.1", "utf-8")
	d.Register("192.0.2.2", "shift-jis")

	t.Run("select utf-8", func(t *testing.T) {
		actual, err := d.Decode("192.0.2.1", []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f})
		if err != nil {
			t.Errorf("error should be nil but: %+v", err)
		}
		expected := "Hello"
		if actual != expected {
			t.Errorf("value should be %+v but: %+v", expected, actual)
		}
	})

	t.Run("select shift-jis", func(t *testing.T) {
		actual, err := d.Decode("192.0.2.2", []byte{0x83, 0x41})
		if err != nil {
			t.Errorf("error should be nil but: %+v", err)
		}
		expected := "ア"
		if actual != expected {
			t.Errorf("value should be %+v but: %+v", expected, actual)
		}
	})

	t.Run("default is utf-8", func(t *testing.T) {
		actual, err := d.Decode("192.0.2.3", []byte{0xe3, 0x81, 0x82})
		if err != nil {
			t.Errorf("error should be nil but: %+v", err)
		}
		expected := "あ"
		if actual != expected {
			t.Errorf("value should be %+v but: %+v", expected, actual)
		}
	})
}

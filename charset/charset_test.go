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

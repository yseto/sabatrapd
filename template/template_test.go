package template

import "testing"

func TestParse(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		format := `{{ addr }} {{ read "IF-MIB::ifDescr" }} is linkup`
		err := Parse(format)
		if err != nil {
			t.Errorf("error should be nil but: %+v", err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		format := `{{ invalid_function "IF-MIB::ifDescr" }} is linkup`
		actual := Parse(format)
		if actual == nil {
			t.Errorf("should return error")
		}
		expected := `template: :1: function "invalid_function" not defined`
		if actual.Error() != expected {
			t.Errorf("error should be %+v but: %+v", expected, actual.Error())
		}
	})
}

func TestExecute(t *testing.T) {
	format := `{{ addr }} {{ read "IF-MIB::ifDescr" }} is linkup`
	pad := map[string]string{
		"IF-MIB::ifDescr": "eth0",
		"IF-MIB::ifIndex": "0",
	}

	actual, err := Execute(format, pad, "192.0.2.1")
	if err != nil {
		t.Errorf("error should be nil but: %+v", err)
	}

	expected := "192.0.2.1 eth0 is linkup"
	if actual != expected {
		t.Errorf("result should be %+v but: %+v", expected, actual)
	}
}

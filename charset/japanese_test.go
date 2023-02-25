package charset

import (
	"testing"
)

func TestTransformShiftJIS(t *testing.T) {
	b := []byte{0xc3, 0xbd, 0xc4, 0xd2, 0xaf, 0xbe, 0xb0, 0xbc, 0xde, 0x82, 0xc5, 0x82, 0xb7, 0x81, 0x42}
	actual, err := transformShiftJIS(b)
	if err != nil {
		t.Errorf("error should be nil but: %+v", err)
	}
	expected := "ﾃｽﾄﾒｯｾｰｼﾞです。"
	if actual != expected {
		t.Errorf("transformShiftJIS: %v should be %v", actual, expected)
	}
}

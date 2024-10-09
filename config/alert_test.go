package config

import (
	"testing"

	"github.com/mackerelio/mackerel-client-go"
)

func TestValidateAlertLevel(t *testing.T) {
	if err := ValidateAlertLevel("unknown"); err != nil {
		t.Error(err)
	}
	if err := ValidateAlertLevel("critical"); err != nil {
		t.Error(err)
	}
	if err := ValidateAlertLevel("warning"); err != nil {
		t.Error(err)
	}

	err := ValidateAlertLevel("foo")
	if err == nil {
		t.Error("invalid result")
	}

	if err.Error() != "alert-level is invalid. foo, valid argument: [unknown critical warning]" {
		t.Error("invalid result")
	}
}

func TestConvertAlertLevel(t *testing.T) {
	if ConvertAlertLevel("unknown") != mackerel.CheckStatusUnknown {
		t.Error("invalid")
	}
	if ConvertAlertLevel("critical") != mackerel.CheckStatusCritical {
		t.Error("invalid")
	}
	if ConvertAlertLevel("warning") != mackerel.CheckStatusWarning {
		t.Error("invalid")
	}
}

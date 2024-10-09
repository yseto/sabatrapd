package config

import (
	"fmt"
	"slices"

	"github.com/mackerelio/mackerel-client-go"
)

const (
	levelUnknown  = "unknown"
	levelCritical = "critical"
	levelWarning  = "warning"
)

func validAlertLevel() []string {
	return []string{levelUnknown, levelCritical, levelWarning}
}

// 無指定の場合は WARNING 入力は ValidateAlertLevel で検査されていること
func ConvertAlertLevel(s string) mackerel.CheckStatus {
	switch s {
	case levelUnknown:
		return mackerel.CheckStatusUnknown
	case levelCritical:
		return mackerel.CheckStatusCritical
	}
	return mackerel.CheckStatusWarning
}

func ValidateAlertLevel(s string) error {
	if s != "" && !slices.Contains(validAlertLevel(), s) {
		return fmt.Errorf("alert-level is invalid. %s, valid argument: %s", s, validAlertLevel())
	}
	return nil
}

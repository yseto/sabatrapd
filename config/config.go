package config

import (
	"cmp"
	"slices"

	"github.com/yseto/sabatrapd/charset"
	"github.com/yseto/sabatrapd/oid"
)

type MIB struct {
	Directory   []string `yaml:"directory"`
	LoadModules []string `yaml:"modules"`
}

type TrapServer struct {
	Address   string `yaml:"addr"`
	Port      string `yaml:"port"`
	Community string `yaml:"community"`
}

type Trap struct {
	Ident       string `yaml:"ident"`
	ParsedIdent []int  `yaml:"-"`
	Format      string `yaml:"format"`
	AlertLevel  string `yaml:"alert-level"` // warning, critical, unknown. default: warning
}

type Mackerel struct {
	ApiKey  string `yaml:"x-api-key"`
	ApiBase string `yaml:"apibase"`
	HostID  string `yaml:"host-id"`
}

type Encoding struct {
	Address string          `yaml:"addr"`
	Charset charset.Charset `yaml:"charset"`
}

type Config struct {
	MIB        *MIB        `yaml:"mib"`
	TrapServer *TrapServer `yaml:"snmp"`
	Trap       []*Trap     `yaml:"trap"`
	Debug      bool        `yaml:"debug"`
	DryRun     bool        `yaml:"dry-run"`
	Mackerel   *Mackerel   `yaml:"mackerel"`
	Encoding   []*Encoding `yaml:"encoding"`
}

func (c *Config) RunningMode() string {
	if c.DryRun {
		return "dry-run"
	}
	return "execute"
}

func (c *Config) SortedTrapRules() ([]*Trap, error) {
	trap := c.Trap
	for i := range trap {
		r, err := oid.Parse(trap[i].Ident)
		if err != nil {
			return nil, err
		}
		trap[i].ParsedIdent = r
	}

	return slices.SortedFunc(slices.Values(trap), func(a, b *Trap) int {
		return cmp.Or(
			cmp.Compare(len(b.ParsedIdent), len(a.ParsedIdent)),
			cmp.Compare(a.Ident, b.Ident),
		)
	}), nil
}

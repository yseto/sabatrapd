package config

import "github.com/yseto/sabatrapd/charset"

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
	Ident  string `yaml:"ident"`
	Format string `yaml:"format"`
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

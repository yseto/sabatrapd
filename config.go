package main

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
	ApiKey string `yaml:"x-api-key"`
	HostID string `yaml:"host-id"`
}

type Config struct {
	MIB        *MIB        `yaml:"mib"`
	TrapServer *TrapServer `yaml:"snmp"`
	Trap       []*Trap     `yaml:"trap"`
	Debug      bool        `yaml:"debug"`
	Mackerel   *Mackerel   `yaml:"mackerel"`
}

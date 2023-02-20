package main

type MIB struct {
	Directory   []string `yaml:"directory"`
	LoadModules []string `yaml:"modules"`
}

type TrapServer struct {
	Address string `yaml:"addr"`
	Port    string `yaml:"port"`
}

type Trap struct {
	Ident  string `yaml:"ident"`
	Format string `yaml:"format"`
}

type Config struct {
	MIB        *MIB        `yaml:"mib"`
	TrapServer *TrapServer `yaml:"snmp"`
	Trap       []*Trap     `yaml:"trap"`
	Debug      bool        `yaml:"debug"`
}

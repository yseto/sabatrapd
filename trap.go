package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"text/template"

	g "github.com/gosnmp/gosnmp"
	"github.com/sleepinggenius2/gosmi/types"
	"gopkg.in/yaml.v3"
)

const SnmpTrapOIDPrefix = ".1.3.6.1.6.3.1.1.4.1"

var mibParser SMI
var c Config

func main() {
	defer func() {
		mibParser.Close()
	}()

	// TODO args.
	f, err := os.ReadFile("config.yaml")

	err = yaml.Unmarshal(f, &c)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	mibParser.Modules = c.MIB.LoadModules
	mibParser.Paths = c.MIB.Directory
	mibParser.Init()

	// template tests.
	funcmap := template.FuncMap{
		"read": func(key string) string {
			return "dummy"
		},
		"addr": func() string {
			return "dummy"
		},
	}

	var tmpl = template.New("").Funcs(funcmap)

	for i := range c.Trap {
		if _, err := tmpl.Parse(c.Trap[i].Format); err != nil {
			log.Fatalln(err)
		}
	}

	trapListner := g.NewTrapListener()
	trapListner.OnNewTrap = trapHandler
	trapListner.Params = g.Default
	if c.Debug {
		trapListner.Params.Logger = g.NewLogger(log.New(os.Stdout, "<GOSNMP DEBUG LOGGER>", 0))
	}

	err = trapListner.Listen(net.JoinHostPort(c.TrapServer.Address, c.TrapServer.Port))
	if err != nil {
		log.Fatalf("error in listen: %s", err)
	}
}

func trapHandler(packet *g.SnmpPacket, addr *net.UDPAddr) {
	// log.Printf("got trapdata from %s\n", addr.IP)

	var pad = make(map[string]string)
	var specificTrapFormat string

	for _, v := range packet.Variables {
		if strings.HasPrefix(v.Name, SnmpTrapOIDPrefix) {
			for i := range c.Trap {
				if strings.HasPrefix(v.Value.(string), c.Trap[i].Ident) {
					specificTrapFormat = c.Trap[i].Format
				}
			}
		}

		var padKey, padValue string
		padKey = v.Name
		node, err := mibParser.FromOID(v.Name)
		if err != nil {
			fmt.Printf("%+v\n", err)
		} else {
			if node != nil {
				padKey = node.Node.RenderQualified()
			}

			if node.Node.Type != nil && node.Node.Type.BaseType == types.BaseTypeEnum {
				i, ok := v.Value.(int)
				if ok {
					padValue = node.Node.Type.Enum.Name(int64(i))
				}
			}
		}
		if padValue == "" {
			switch v.Type {
			case g.OctetString:
				b := v.Value.([]byte)
				padValue = string(b)
				// fmt.Printf("OID: %s, string: %s\n", v.Name, string(b))
			case g.ObjectIdentifier:
				valNode, err := mibParser.FromOID(v.Value.(string))
				if err != nil {
					fmt.Printf("%+v\n", err)
					padValue = v.Value.(string)
				} else {
					padValue = valNode.Node.Name
				}
				// fmt.Printf("OID: %s, value: %s ObjectIdentifier: %s\n", v.Name, v.Value.(string), valNode.Node.Name)
			default:
				// fmt.Printf("trap: %+v\n", v)
				padValue = fmt.Sprintf("%v", v.Value)
			}
		}

		if padKey != "" && padValue != "" {
			pad[padKey] = padValue
		}
	}

	if specificTrapFormat == "" {
		if c.Debug {
			log.Printf("skip because nothing template : %+v\n", pad)
		}
		return
	}

	funcmap := template.FuncMap{
		"read": func(key string) string {
			return fmt.Sprintf("%s", pad[key])
		},
		"addr": func() string {
			return addr.IP.String()
		},
	}

	// fmt.Printf("%+v\n", pad)

	var tpl = template.New("").Funcs(funcmap)

	if err := template.Must(tpl.Parse(specificTrapFormat)).Execute(os.Stdout, pad); err != nil {
		log.Println(err)
	}
}

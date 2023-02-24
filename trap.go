package main

import (
	"bufio"
	"bytes"
	"container/list"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"text/template"
	"time"

	g "github.com/gosnmp/gosnmp"
	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/sleepinggenius2/gosmi/types"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

const SnmpTrapOIDPrefix = ".1.3.6.1.6.3.1.1.4.1"

var mibParser SMI
var c Config
var buffers = list.New()
var mutex = &sync.Mutex{}
var CharsetMap = make(map[string]Charset)

func main() {
	// TODO args.
	f, err := os.ReadFile("config.yaml")

	// load config.
	err = yaml.Unmarshal(f, &c)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// init mib parser
	mibParser.Modules = c.MIB.LoadModules
	mibParser.Paths = c.MIB.Directory
	mibParser.Init()
	defer mibParser.Close()

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

	// encoding tests.
	encodings := []Charset{CharsetShiftJis, CharsetUTF8}
	for i := range c.Encoding {
		if net.ParseIP(c.Encoding[i].Address) == nil {
			log.Fatalf("cant parse ip : %q", c.Encoding[i].Address)
		}
		ok := false
		for j := range encodings {
			if encodings[j] == c.Encoding[i].Charset {
				ok = true
				CharsetMap[c.Encoding[i].Address] = c.Encoding[i].Charset
			}
		}
		if !ok {
			log.Fatalf("charset is missing. %q", c.Encoding[i].Charset)
		}
	}

	// trapListener
	trapListener := g.NewTrapListener()
	trapListener.OnNewTrap = trapHandler
	trapListener.Params = g.Default
	if c.Debug {
		trapListener.Params.Logger = g.NewLogger(log.New(os.Stdout, "<GOSNMP DEBUG LOGGER>", 0))
	}

	client := mackerel.NewClient(c.Mackerel.ApiKey)

	wg := sync.WaitGroup{}

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

	wg.Add(1)
	go func() {
		err = trapListener.Listen(net.JoinHostPort(c.TrapServer.Address, c.TrapServer.Port))
		if err != nil {
			log.Fatalf("error in listen: %s", err)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-t.C:
				sendToMackerel(ctx, client, c.Mackerel.HostID)

			case <-ctx.Done():
				trapListener.Close()
				log.Println("trapListener is close.")
				log.Println("cancellation from context:", ctx.Err())
				return
			}
		}
	}()
	log.Println("initialized.")
	wg.Wait()
}

func trapHandler(packet *g.SnmpPacket, addr *net.UDPAddr) {
	// log.Printf("got trapdata from %s\n", addr.IP)

	if c.TrapServer.Community != packet.Community {
		if c.Debug {
			log.Printf("invalid community: expected '%q', but received '%q'", c.TrapServer.Community, packet.Community)
		}
		return
	}

	var pad = make(map[string]string)
	var specificTrapFormat string
	var occurredAt = time.Now().Unix()

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
				switch CharsetMap[addr.IP.String()] {
				case CharsetShiftJis:
					padValue, err = transformShiftJIS(b)
					if err != nil {
						fmt.Printf("%+v\n", err)
						padValue = "<can not decode>"
					}
				case CharsetUTF8:
					fallthrough
				default:
					padValue = string(b)
				}
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

	var wr bytes.Buffer
	if err := template.Must(tpl.Parse(specificTrapFormat)).Execute(&wr, pad); err != nil {
		log.Println(err)
	}

	mutex.Lock()
	buffers.PushBack(mackerelCheck{
		OccurredAt: occurredAt,
		Addr:       addr.IP.String(),
		Message:    wr.String(),
	})
	mutex.Unlock()
}

func transformShiftJIS(b []byte) (string, error) {
	scanner := bufio.NewScanner(transform.NewReader(bytes.NewBuffer(b), japanese.ShiftJIS.NewDecoder()))
	var str string
	for scanner.Scan() {
		str += scanner.Text()
	}
	return str, scanner.Err()
}

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/yseto/sabatrapd/charset"
	"github.com/yseto/sabatrapd/config"
	"github.com/yseto/sabatrapd/notification"
	"github.com/yseto/sabatrapd/smi"
	"github.com/yseto/sabatrapd/template"

	g "github.com/gosnmp/gosnmp"
	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/sleepinggenius2/gosmi/types"
	"gopkg.in/yaml.v3"
)

const SnmpTrapOIDPrefix = ".1.3.6.1.6.3.1.1.4.1"

var mibParser smi.SMI
var c config.Config
var decoder = charset.NewDecoder()
var queue *notification.Queue

func main() {
	// TODO args.
	f, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalln(err)
	}

	// load config.
	err = yaml.Unmarshal(f, &c)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// init mib parser
	mibParser.Modules = c.MIB.LoadModules
	mibParser.Paths = c.MIB.Directory
	mibParser.Init()
	defer mibParser.Close()

	// template tests.
	for i := range c.Trap {
		if err := template.Parse(c.Trap[i].Format); err != nil {
			log.Fatalln(err)
		}
	}

	// encoding tests.
	for i := range c.Encoding {
		if net.ParseIP(c.Encoding[i].Address) == nil {
			log.Fatalf("can't parse ip : %q", c.Encoding[i].Address)
		}
		if err = decoder.Register(c.Encoding[i].Address, c.Encoding[i].Charset); err != nil {
			log.Fatalln(err)
		}
	}

	if c.Mackerel.ApiKey == "" {
		log.Fatalf("x-api-key isn't defined.")
	}
	if c.Mackerel.HostID == "" {
		log.Fatalf("host-id isn't defined.")
	}

	var client *mackerel.Client
	if c.Mackerel.ApiBase == "" {
		client = mackerel.NewClient(c.Mackerel.ApiKey)
	} else {
		client, err = mackerel.NewClientWithOptions(c.Mackerel.ApiKey, c.Mackerel.ApiBase, false)
		if err != nil {
			log.Fatalf("invalid apibase: %s", err)
		}
	}

	_, err = client.FindHost(c.Mackerel.HostID)
	if err != nil {
		log.Fatalf("Either x-api-key or host-id is invalid.\n%s", err)
	}

	queue = notification.NewQueue(client, c.Mackerel.HostID)

	// trapListener
	trapListener := g.NewTrapListener()
	trapListener.OnNewTrap = trapHandler
	trapListener.Params = g.Default
	if c.Debug {
		trapListener.Params.Logger = g.NewLogger(log.New(os.Stdout, "<GOSNMP DEBUG LOGGER>", 0))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

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
				queue.Dequeue(ctx)

			case <-ctx.Done():
				trapListener.Close()
				log.Println("trapListener is closed.")
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
			log.Printf("invalid community: expected %q, but received %q", c.TrapServer.Community, packet.Community)
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
				padValue, err = decoder.Decode(addr.IP.String(), b)
				if err != nil {
					fmt.Printf("%+v\n", err)
					padValue = "<cannot decode>"
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

	message, err := template.Execute(specificTrapFormat, pad, addr.IP.String())
	if err != nil {
		log.Println(err)
		return
	}

	queue.Enqueue(notification.Item{
		OccurredAt: occurredAt,
		Addr:       addr.IP.String(),
		Message:    message,
	})
}

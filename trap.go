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

func main() {
	// TODO args.
	f, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalln(err)
	}

	// load config.
	var conf config.Config
	err = yaml.Unmarshal(f, &conf)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// init mib parser
	var mibParser smi.SMI
	mibParser.Modules = conf.MIB.LoadModules
	mibParser.Paths = conf.MIB.Directory
	mibParser.Init()
	defer mibParser.Close()

	// template tests.
	for i := range conf.Trap {
		if err := template.Parse(conf.Trap[i].Format); err != nil {
			log.Fatalln(err)
		}
	}

	decoder := charset.NewDecoder()

	// encoding tests.
	for i := range conf.Encoding {
		if net.ParseIP(conf.Encoding[i].Address) == nil {
			log.Fatalf("can't parse ip : %q", conf.Encoding[i].Address)
		}
		if err = decoder.Register(conf.Encoding[i].Address, conf.Encoding[i].Charset); err != nil {
			log.Fatalln(err)
		}
	}

	if conf.Mackerel.ApiKey == "" {
		log.Fatalf("x-api-key isn't defined.")
	}
	if conf.Mackerel.HostID == "" {
		log.Fatalf("host-id isn't defined.")
	}

	var client *mackerel.Client
	if conf.Mackerel.ApiBase == "" {
		client = mackerel.NewClient(conf.Mackerel.ApiKey)
	} else {
		client, err = mackerel.NewClientWithOptions(conf.Mackerel.ApiKey, conf.Mackerel.ApiBase, false)
		if err != nil {
			log.Fatalf("invalid apibase: %s", err)
		}
	}

	_, err = client.FindHost(conf.Mackerel.HostID)
	if err != nil {
		log.Fatalf("Either x-api-key or host-id is invalid.\n%s", err)
	}

	queue := notification.NewQueue(client, conf.Mackerel.HostID)

	handler := &Handler{
		&conf,
		queue,
		&mibParser,
		decoder,
	}

	// trapListener
	trapListener := g.NewTrapListener()
	trapListener.OnNewTrap = handler.OnNewTrap
	trapListener.Params = g.Default
	if conf.Debug {
		trapListener.Params.Logger = g.NewLogger(log.New(os.Stdout, "<GOSNMP DEBUG LOGGER>", 0))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	wg := sync.WaitGroup{}

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

	wg.Add(1)
	go func() {
		err = trapListener.Listen(net.JoinHostPort(conf.TrapServer.Address, conf.TrapServer.Port))
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

type Handler struct {
	Config    *config.Config
	Queue     *notification.Queue
	MibParser *smi.SMI
	Decoder   *charset.Decoder
}

func (h *Handler) OnNewTrap(packet *g.SnmpPacket, addr *net.UDPAddr) {
	// log.Printf("got trapdata from %s\n", addr.IP)
	config := h.Config

	if config.TrapServer.Community != packet.Community {
		if config.Debug {
			log.Printf("invalid community: expected %q, but received %q", config.TrapServer.Community, packet.Community)
		}
		return
	}

	var pad = make(map[string]string)
	var specificTrapFormat string
	var occurredAt = time.Now().Unix()

	for _, v := range packet.Variables {
		if strings.HasPrefix(v.Name, SnmpTrapOIDPrefix) {
			for i := range config.Trap {
				if strings.HasPrefix(v.Value.(string), config.Trap[i].Ident) {
					specificTrapFormat = config.Trap[i].Format
				}
			}
		}

		var padKey, padValue string
		padKey = v.Name
		node, err := h.MibParser.FromOID(v.Name)
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
				padValue, err = h.Decoder.Decode(addr.IP.String(), b)
				if err != nil {
					fmt.Printf("%+v\n", err)
					padValue = "<cannot decode>"
				}
				// fmt.Printf("OID: %s, string: %s\n", v.Name, string(b))
			case g.ObjectIdentifier:
				valNode, err := h.MibParser.FromOID(v.Value.(string))
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
		if config.Debug {
			log.Printf("skip because nothing template : %+v\n", pad)
		}
		return
	}

	message, err := template.Execute(specificTrapFormat, pad, addr.IP.String())
	if err != nil {
		log.Println(err)
		return
	}

	h.Queue.Enqueue(notification.Item{
		OccurredAt: occurredAt,
		Addr:       addr.IP.String(),
		Message:    message,
	})
}

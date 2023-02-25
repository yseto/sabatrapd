package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/yseto/sabatrapd/charset"
	"github.com/yseto/sabatrapd/config"
	"github.com/yseto/sabatrapd/handler"
	"github.com/yseto/sabatrapd/notification"
	"github.com/yseto/sabatrapd/smi"
	"github.com/yseto/sabatrapd/template"

	g "github.com/gosnmp/gosnmp"
	mackerel "github.com/mackerelio/mackerel-client-go"
	"gopkg.in/yaml.v3"
)

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

	handle := &handler.Handler{
		Config:    &conf,
		Queue:     queue,
		MibParser: &mibParser,
		Decoder:   decoder,
	}

	// trapListener
	trapListener := g.NewTrapListener()
	trapListener.OnNewTrap = handle.OnNewTrap
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

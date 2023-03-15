package main

import (
	"context"
	"flag"
	"fmt"
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

var configFilename string
var dryRun bool

func init() {
	flag.StringVar(&configFilename, "conf", "sabatrapd.yml", "config `filename`")
	flag.BoolVar(&dryRun, "dry-run", false, "dry run mode")
}

func main() {
	flag.Parse()

	f, err := os.ReadFile(configFilename)
	if err != nil {
		log.Fatalln(err)
	}

	// load config.
	var conf config.Config
	err = yaml.Unmarshal(f, &conf)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	// merge dry-run argument.
	conf.DryRun = (conf.DryRun || dryRun)

	// init mib parser
	var mibParser smi.SMI
	if conf.MIB != nil {
		mibParser.Modules = conf.MIB.LoadModules
		mibParser.Paths = conf.MIB.Directory
	}
	err = mibParser.Init()
	if err != nil {
		log.Println(err)
	}
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

	var client notification.Client
	var hostid string
	if !conf.DryRun {
		client, err = checkMackerelConfig(conf.Mackerel)
		if err != nil {
			log.Fatalln(err)
		}
		hostid = conf.Mackerel.HostID
	} else {
		hostid = ""
	}

	queue := notification.NewQueue(client, hostid)

	handle := &handler.Handler{
		Config:    &conf,
		Queue:     queue,
		MibParser: &mibParser,
		Decoder:   decoder,
	}

	// trapListener
	if conf.TrapServer == nil || conf.TrapServer.Address == "" || conf.TrapServer.Port == "" {
		log.Fatalln("either addr or port isn't defined")
	}

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
	log.Printf("initialized. %s mode\n", conf.RunningMode())
	wg.Wait()
}

func checkMackerelConfig(conf *config.Mackerel) (*mackerel.Client, error) {
	if conf == nil || conf.ApiKey == "" {
		return nil, fmt.Errorf("x-api-key isn't defined")
	}
	if conf.HostID == "" {
		return nil, fmt.Errorf("host-id isn't defined")
	}

	var client *mackerel.Client
	var err error

	if conf.ApiBase == "" {
		client = mackerel.NewClient(conf.ApiKey)
	} else {
		client, err = mackerel.NewClientWithOptions(conf.ApiKey, conf.ApiBase, false)
		if err != nil {
			return nil, fmt.Errorf("invalid apibase: %s", err)
		}
	}

	_, err = client.FindHost(conf.HostID)
	if err != nil {
		return nil, fmt.Errorf("either x-api-key or host-id is invalid: %s", err)
	}
	return client, nil
}

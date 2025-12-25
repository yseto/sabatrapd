package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/mackerelio-labs/sabatrapd/charset"
	"github.com/mackerelio-labs/sabatrapd/config"
	"github.com/mackerelio-labs/sabatrapd/handler"
	"github.com/mackerelio-labs/sabatrapd/notification"
	"github.com/mackerelio-labs/sabatrapd/smi"
	"github.com/mackerelio-labs/sabatrapd/template"

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
		slog.Error(err.Error())
		os.Exit(1)
	}

	// load config.
	var conf config.Config
	err = yaml.Unmarshal(f, &conf)
	if err != nil {
		slog.Error("failed read config", "error", err)
		os.Exit(1)
	}
	// merge dry-run argument.
	conf.DryRun = (conf.DryRun || dryRun)

	logLevel, ok := conf.GetLogLebel()
	if !ok {
		slog.Error("failed parse log level")
		os.Exit(1)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// init mib parser
	var mibParser smi.SMI
	if conf.MIB != nil {
		mibParser.Modules = conf.MIB.LoadModules
		mibParser.Paths = conf.MIB.Directory
	}
	err = mibParser.Init()
	if err != nil {
		slog.Error("failed init mibParser", "error", err)
		os.Exit(1)
	}
	defer mibParser.Close()

	for i := range conf.Trap {
		// template tests.
		if err := template.Parse(conf.Trap[i].Format); err != nil {
			slog.Error("failed parse template", "error", err)
			os.Exit(1)
		}
		// validate alert-level
		if err := config.ValidateAlertLevel(conf.Trap[i].AlertLevel); err != nil {
			slog.Error("failed validate alert level", "error", err)
			os.Exit(1)
		}
	}

	decoder := charset.NewDecoder()

	// encoding tests.
	for i := range conf.Encoding {
		if net.ParseIP(conf.Encoding[i].Address) == nil {
			slog.Error("can't parse ip", "address", conf.Encoding[i].Address)
			os.Exit(1)
		}
		if err = decoder.Register(conf.Encoding[i].Address, conf.Encoding[i].Charset); err != nil {
			slog.Error("encoding error", "error", err)
			os.Exit(1)
		}
	}

	var client notification.Client
	var hostid string
	if !conf.DryRun {
		client, err = checkMackerelConfig(conf.Mackerel)
		if err != nil {
			slog.Error("mackerel config error", "error", err)
			os.Exit(1)
		}
		hostid = conf.Mackerel.HostID
	} else {
		hostid = ""
	}

	queue := notification.NewQueue(client, hostid)

	traps, err := conf.SortedTrapRules()
	if err != nil {
		slog.Error("failed parsed config", "error", err)
		os.Exit(1)
	}
	handle := &handler.Handler{
		Community: conf.TrapServer.Community,
		Traps:     traps,

		Queue:     queue,
		MibParser: &mibParser,
		Decoder:   decoder,
	}

	// trapListener
	if conf.TrapServer == nil || conf.TrapServer.Address == "" || conf.TrapServer.Port == "" {
		slog.Error("either addr or port isn't defined")
		os.Exit(1)
	}

	trapListener := g.NewTrapListener()
	trapListener.OnNewTrap = handle.OnNewTrap
	trapListener.Params = g.Default
	trapListener.Params.Logger = g.NewLogger(trapListenerLogger{})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	wg := sync.WaitGroup{}

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

	wg.Add(1)
	go func() {
		err = trapListener.Listen(net.JoinHostPort(conf.TrapServer.Address, conf.TrapServer.Port))
		if err != nil {
			slog.Error("error in listen", "error", err)
			os.Exit(1)
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
				slog.Info("trapListener is closed.")
				return
			}
		}
	}()
	slog.Info("initialized", "mode", conf.RunningMode())
	wg.Wait()
}

type trapListenerLogger struct{}

func (trapListenerLogger) Print(v ...interface{}) {
	slog.Debug(fmt.Sprint(v...))
}
func (trapListenerLogger) Printf(format string, v ...interface{}) {
	slog.Debug(fmt.Sprintf(format, v...))
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

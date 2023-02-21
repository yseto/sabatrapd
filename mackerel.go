package main

import (
	"context"
	"fmt"
	"log"

	mackerel "github.com/mackerelio/mackerel-client-go"
)

type mackerelCheck struct {
	OccurredAt int64
	Addr       string
	Message    string
}

func sendToMackerel(ctx context.Context, client *mackerel.Client, hostId string) {
	if buffers.Len() == 0 {
		return
	}

	e := buffers.Front()
	// log.Infof("send current value: %#v", e.Value)
	// log.Infof("buffers len: %d", buffers.Len())

	v := e.Value.(mackerelCheck)

	// TODO: message length...
	reports := []*mackerel.CheckReport{
		{
			Source:     mackerel.NewCheckSourceHost(hostId),
			Status:     mackerel.CheckStatusWarning,
			Name:       fmt.Sprintf("snmptrap %s", v.Addr),
			Message:    v.Message,
			OccurredAt: v.OccurredAt,
		},
	}
	err := client.PostCheckReports(&mackerel.CheckReports{Reports: reports})
	if err != nil {
		log.Println(err)
		return
	} else {
		log.Printf("mackerel success: %q %q", v.Addr, v.Message)
	}
	mutex.Lock()
	buffers.Remove(e)
	mutex.Unlock()
}

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	mackerel "github.com/mackerelio/mackerel-client-go"
)

type mackerelCheck struct {
	Addr    string
	Message string
}

func sendToMackerel(ctx context.Context, client *mackerel.Client, hostId string) {
	if buffers.Len() == 0 {
		return
	}

	e := buffers.Front()
	// log.Infof("send current value: %#v", e.Value)
	// log.Infof("buffers len: %d", buffers.Len())

	v := e.Value.(mackerelCheck)

	reports := []*mackerel.CheckReport{
		{
			Source:     mackerel.NewCheckSourceHost(hostId),
			Status:     mackerel.CheckStatusWarning,
			Name:       fmt.Sprintf("snmptrap %s", v.Addr),
			Message:    v.Message,
			OccurredAt: time.Now().Unix(),
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

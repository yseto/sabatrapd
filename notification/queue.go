package notification

import (
	"container/list"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"unicode/utf8"

	"github.com/mackerelio/mackerel-client-go"

	"github.com/yseto/sabatrapd/config"
)

type Client interface {
	PostCheckReports(crs *mackerel.CheckReports) error
}

type Queue struct {
	q      *list.List
	m      sync.Mutex
	client Client
	hostId string
}

type Item struct {
	OccurredAt int64
	Addr       string
	Message    string
	AlertLevel string
}

// NewQueue is needed mackerel client, host id.
func NewQueue(client Client, hostId string) *Queue {
	return &Queue{
		q:      list.New(),
		client: client,
		hostId: hostId,
	}
}

func (q *Queue) Enqueue(item Item) {
	q.m.Lock()
	q.q.PushBack(item)
	q.m.Unlock()
}

func (q *Queue) Dequeue(ctx context.Context) {
	if q.q.Len() == 0 {
		return
	}

	e := q.q.Front()
	item := e.Value.(Item)
	if q.client == nil {
		slog.Info("receive", "addr", item.Addr, "message", item.Message, "alertLevel", config.ConvertAlertLevel(item.AlertLevel))
	} else {
		err := q.send(item)
		if err != nil {
			slog.Error("send error", "error", err.Error())
			return
		}
	}
	q.m.Lock()
	q.q.Remove(e)
	q.m.Unlock()
}

const msgLengthLimit = 1024

func (q *Queue) send(item Item) error {
	message := item.Message
	if utf8.RuneCountInString(message) > msgLengthLimit {
		message = string([]rune(message)[0:msgLengthLimit])
	}

	reports := []*mackerel.CheckReport{
		{
			Source:     mackerel.NewCheckSourceHost(q.hostId),
			Status:     config.ConvertAlertLevel(item.AlertLevel),
			Name:       fmt.Sprintf("sabatrapd %s", item.Addr),
			Message:    message,
			OccurredAt: item.OccurredAt,
		},
	}
	err := q.client.PostCheckReports(&mackerel.CheckReports{Reports: reports})
	if err != nil {
		return err
	}
	slog.Info("mackerel success", "addr", item.Addr, "message", item.Message)
	return nil
}

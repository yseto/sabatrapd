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
	q       *list.List
	m       sync.Mutex
	client  Client
	hostId  string
	maxSize int
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
		q:       list.New(),
		client:  client,
		hostId:  hostId,
		maxSize: 1000, // デフォルトで1000件まで
	}
}

func (q *Queue) Enqueue(item Item) {
	q.m.Lock()
	defer q.m.Unlock()
	
	// キューサイズが上限に達している場合、古いアイテムを削除
	if q.q.Len() >= q.maxSize {
		// 古いアイテムを削除
		if front := q.q.Front(); front != nil {
			q.q.Remove(front)
			slog.Warn("queue full, removing oldest item", "queueSize", q.q.Len(), "maxSize", q.maxSize)
		}
	}
	
	q.q.PushBack(item)
}

func (q *Queue) Dequeue(ctx context.Context) {
	// バッチ処理: 最大10件まで一度に処理
	batchSize := 10
	processed := 0
	
	for processed < batchSize {
		q.m.Lock()
		if q.q.Len() == 0 {
			q.m.Unlock()
			break
		}
		
		e := q.q.Front()
		item := e.Value.(Item)
		q.q.Remove(e)
		q.m.Unlock()
		
		if q.client == nil {
			slog.Info("receive", "addr", item.Addr, "message", item.Message, "alertLevel", config.ConvertAlertLevel(item.AlertLevel))
		} else {
			err := q.send(item)
			if err != nil {
				slog.Warn("send error", "error", err.Error())
				// エラーが発生した場合、このアイテムは破棄する（無限リトライを避ける）
				continue
			}
		}
		processed++
	}
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

// GetQueueSize returns the current queue size
func (q *Queue) GetQueueSize() int {
	q.m.Lock()
	defer q.m.Unlock()
	return q.q.Len()
}

// SetMaxSize sets the maximum queue size
func (q *Queue) SetMaxSize(size int) {
	q.m.Lock()
	defer q.m.Unlock()
	q.maxSize = size
}

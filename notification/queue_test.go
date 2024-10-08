package notification

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/mackerelio/mackerel-client-go"
)

type mockClient struct {
	reports *mackerel.CheckReports
}

func (m *mockClient) PostCheckReports(crs *mackerel.CheckReports) error {
	m.reports = crs
	return nil
}

func TestRoundOffMessage(t *testing.T) {
	log.SetOutput(io.Discard)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}()

	client := &mockClient{}

	msg := strings.Repeat("a", 2048)

	q := NewQueue(client, "")
	q.Enqueue(Item{OccurredAt: 1, Addr: "192.0.2.1", Message: msg})
	ctx := context.Background()
	q.Dequeue(ctx)

	report := client.reports.Reports[0]

	if len(report.Message) != 1024 {
		t.Error("invalid round Message")
	}
}

func TestQueue(t *testing.T) {
	client := &mockClient{}
	q := NewQueue(client, "")
	q.Enqueue(Item{OccurredAt: 1, Addr: "192.0.2.1", Message: "message"})
	ctx := context.Background()
	q.Dequeue(ctx)

	report := client.reports.Reports[0]

	if report.Name != "sabatrapd 192.0.2.1" {
		t.Error("invalid Name")
	}
	if report.Status != "WARNING" {
		t.Error("invalid Status")
	}
	if report.Message != "message" {
		t.Error("invalid Message")
	}
	if report.OccurredAt != 1 {
		t.Error("invalid OccurredAt")
	}
}

func DryRunQueue() {
	q := NewQueue(nil, "")
	q.Enqueue(Item{OccurredAt: 1, Addr: "192.0.2.1", Message: "message"})

	ctx := context.Background()
	q.Dequeue(ctx)
}

func TestDryRunQueue(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}()
	DryRunQueue()
	actual := buf.String()
	expected := `receive "192.0.2.1" "message" "WARNING"` + "\n"
	if actual != expected {
		t.Errorf("log is invalid. get %q, want %q", actual, expected)
	}
}

type mockErrorClient struct {
}

func (m *mockErrorClient) PostCheckReports(crs *mackerel.CheckReports) error {
	return fmt.Errorf("error %s", crs.Reports[0].Message)
}

func QueueClientError(t *testing.T) {
	client := &mockErrorClient{}
	q := NewQueue(client, "")
	q.Enqueue(Item{OccurredAt: 1, Addr: "192.0.2.1", Message: "message"})
	ctx := context.Background()
	q.Dequeue(ctx)

	actual := q.q.Len()
	expected := 1
	if actual != expected {
		t.Errorf("queue length is invalid. want %d, get %d", expected, actual)
	}
}

func TestQueueClientError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}()
	QueueClientError(t)
	actual := buf.String()
	expected := `error message` + "\n"
	if actual != expected {
		t.Errorf("log is invalid. get %q, want %q", actual, expected)
	}
}

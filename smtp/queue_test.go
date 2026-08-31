package smtp

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestQueueEnqueueAndDequeuePersistsMessage(t *testing.T) {
	queue, err := NewQueue(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := Message{From: "from@example.com", To: "to@example.com", Data: []byte("Subject: test\r\n\r\nhello")}
	id, err := queue.Enqueue(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := queue.Dequeue()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || got.Message.From != want.From || string(got.Message.Data) != string(want.Data) {
		t.Fatalf("got %+v", got)
	}
	if err := queue.Ack(got); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(queue.dir, id+".json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("queue item still exists: %v", err)
	}
}

func TestWorkerRetriesFailedDelivery(t *testing.T) {
	queue, err := NewQueue(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = queue.Enqueue(Message{From: "from@example.com", To: "to@example.com", Data: []byte("body")}); err != nil {
		t.Fatal(err)
	}
	attempts := 0
	worker := Worker{Queue: queue, Delivery: deliveryFunc(func(Message) error { attempts++; return errors.New("temporary") })}
	if err := worker.ProcessOnce(); err == nil {
		t.Fatal("expected delivery failure")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
	if _, err := queue.Dequeue(); err != nil {
		t.Fatalf("message was not retained for retry: %v", err)
	}
}

func TestQueueDoesNotReturnRetryBeforeDue(t *testing.T) {
	queue, err := NewQueue(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = queue.Enqueue(Message{From: "from@example.com", To: "to@example.com", Data: []byte("body")}); err != nil {
		t.Fatal(err)
	}
	item, err := queue.Dequeue()
	if err != nil {
		t.Fatal(err)
	}
	if err := queue.Retry(item); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.DequeueReady(time.Now()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retry was returned before due time: %v", err)
	}
	if _, err := queue.DequeueReady(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("retry was not returned after due time: %v", err)
	}
}

type deliveryFunc func(Message) error

func (f deliveryFunc) Deliver(m Message) error { return f(m) }

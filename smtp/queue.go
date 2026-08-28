package smtp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Delivery interface{ Deliver(Message) error }
type Queue struct{ dir string }
type queuedMessage struct {
	ID          string
	Message     Message
	Attempts    int
	NextAttempt time.Time
}
type QueueItem = queuedMessage

func NewQueue(dir string) (*Queue, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Queue{dir: dir}, nil
}
func (q *Queue) Enqueue(m Message) (string, error) {
	id := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	item := queuedMessage{ID: id, Message: m}
	data, err := json.Marshal(item)
	if err != nil {
		return "", err
	}
	tmp := filepath.Join(q.dir, "."+id+".tmp")
	path := filepath.Join(q.dir, id+".json")
	if err = os.WriteFile(tmp, data, 0600); err != nil {
		return "", err
	}
	if err = os.Rename(tmp, path); err != nil {
		return "", err
	}
	return id, nil
}
func (q *Queue) Dequeue() (queuedMessage, error) {
	entries, err := os.ReadDir(q.dir)
	if err != nil {
		return queuedMessage{}, err
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return queuedMessage{}, os.ErrNotExist
	}
	data, err := os.ReadFile(filepath.Join(q.dir, names[0]))
	if err != nil {
		return queuedMessage{}, err
	}
	var item queuedMessage
	err = json.Unmarshal(data, &item)
	return item, err
}
func (q *Queue) Ack(item queuedMessage) error {
	return os.Remove(filepath.Join(q.dir, item.ID+".json"))
}
func (q *Queue) Retry(item queuedMessage) error {
	item.Attempts++
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(q.dir, item.ID+".json"), data, 0600)
}

type Worker struct {
	Queue    *Queue
	Delivery Delivery
}

func (w Worker) ProcessOnce() error {
	item, err := w.Queue.Dequeue()
	if err != nil {
		return err
	}
	if err = w.Delivery.Deliver(item.Message); err != nil {
		_ = w.Queue.Retry(item)
		return err
	}
	return w.Queue.Ack(item)
}

type RelayDelivery struct {
	Host               string
	Port               int
	Username, Password string
	From               string
}

func (r RelayDelivery) Deliver(m Message) error {
	if r.Host == "" || r.Port == 0 {
		return errors.New("SMTP relay is not configured")
	}
	from := m.From
	if r.From != "" {
		from = r.From
	}
	addr := net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
	var auth smtp.Auth
	if r.Username != "" {
		auth = smtp.PlainAuth("", r.Username, r.Password, r.Host)
	}
	return smtp.SendMail(addr, auth, from, []string{m.To}, m.Data)
}

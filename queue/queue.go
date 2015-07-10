package queue

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/consul/api"
)

var (
	ErrUpdatedFromOther = errors.New("Failed to write by race condition, will wait and retry")
)

type Item struct {
	Name    string
	Trigger string
}

type Queue struct {
	Client *api.Client
	Key    string
}

func init() {
	rand.Seed(time.Now().Unix())
}

func (q *Queue) EnQueue(item Item) error {
	for {
		err := q.enQueue(item)
		if err != ErrUpdatedFromOther {
			return err
		}

		fmt.Println("[Warn]", ErrUpdatedFromOther)
		time.Sleep(time.Duration(rand.Intn(1000)+1000) * time.Millisecond)
	}
}

func (q *Queue) DeQueue() (*Item, error) {
	for {
		item, err := q.deQueue()
		if err != ErrUpdatedFromOther {
			return item, err
		}

		fmt.Println("[Warn]", ErrUpdatedFromOther)
		time.Sleep(time.Duration(rand.Intn(1000)+1000) * time.Millisecond)
	}
}

func (q *Queue) enQueue(item Item) error {
	var items []Item

	entry, _, err := q.Client.KV().Get(q.Key, nil)
	if err != nil {
		return err
	}

	if entry == nil {
		entry = &api.KVPair{Key: q.Key}
	}

	dec := json.NewDecoder(bytes.NewReader(entry.Value))
	dec.Decode(&items)
	items = append(items, item)

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.Encode(&items)

	entry.Value = b.Bytes()

	if result, _, _ := q.Client.KV().CAS(entry, nil); !result {
		return ErrUpdatedFromOther
	}
	return nil
}

func (q *Queue) deQueue() (*Item, error) {
	var items []Item

	entry, _, err := q.Client.KV().Get(q.Key, nil)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	dec := json.NewDecoder(bytes.NewReader(entry.Value))
	dec.Decode(&items)

	if len(items) == 0 {
		return nil, nil
	}

	item := items[0]
	items = items[1:]
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.Encode(&items)

	entry.Value = b.Bytes()

	if result, _, _ := q.Client.KV().CAS(entry, nil); !result {
		return nil, ErrUpdatedFromOther
	}
	return &item, nil
}

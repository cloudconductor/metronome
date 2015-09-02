package queue

import (
	"encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

var (
	ErrUpdatedFromOther = errors.New("Failed to write by race condition, will wait and retry")
)

type Queue struct {
	Client *api.Client
	Key    string
}

func init() {
	rand.Seed(time.Now().Unix())
}

//	Enqueue item to the queue
//	If conflict other process, wait random interval and retry it
func (q *Queue) EnQueue(item interface{}) error {
	for {
		if err := q.enQueue(item); err != ErrUpdatedFromOther {
			return err
		}

		log.Warn(ErrUpdatedFromOther)
		time.Sleep(time.Duration(rand.Intn(1000)+1000) * time.Millisecond)
	}
}

//	Dequeue item from the queue
//	If conflict other process, wait random interval and retry it
func (q *Queue) DeQueue(item interface{}) (error, bool) {
	for {
		if err, found := q.deQueue(item); err != ErrUpdatedFromOther {
			return err, found
		}

		log.Warn(ErrUpdatedFromOther)
		time.Sleep(time.Duration(rand.Intn(1000)+1000) * time.Millisecond)
	}
}

func (q *Queue) enQueue(item interface{}) error {
	var items []interface{}

	//	Get current items from consul KVS
	entry, _, err := q.Client.KV().Get(q.Key, nil)
	if err != nil {
		return err
	}
	//	Create empty value when first time
	if entry == nil {
		entry = &api.KVPair{Key: q.Key}
	}
	if len(entry.Value) > 0 {
		if err := json.Unmarshal(entry.Value, &items); err != nil {
			return err
		}
	}

	//	Add item to the last of items and store new items to consul KVS
	items = append(items, item)
	entry.Value, err = json.Marshal(items)
	if err != nil {
		return err
	}
	if result, _, _ := q.Client.KV().CAS(entry, nil); !result {
		return ErrUpdatedFromOther
	}
	return nil
}

func (q *Queue) deQueue(item interface{}) (error, bool) {
	var items []interface{}

	//	Get current items from consul KVS
	entry, _, err := q.Client.KV().Get(q.Key, nil)
	if err != nil {
		return err, false
	}
	if entry == nil {
		return nil, false
	}
	if len(entry.Value) > 0 {
		if err := json.Unmarshal(entry.Value, &items); err != nil {
			return err, false
		}
	}

	//	return nil if queue is empty
	if len(items) == 0 {
		return nil, false
	}

	//	Get first item from the items
	d, err := json.Marshal(items[0])
	if err != nil {
		return err, false
	}
	if err := json.Unmarshal(d, &item); err != nil {
		return err, false
	}

	//	Remove first item and store new items to consul KVS
	items = items[1:]
	entry.Value, err = json.Marshal(items)
	if err != nil {
		return err, false
	}
	if result, _, _ := q.Client.KV().CAS(entry, nil); !result {
		return ErrUpdatedFromOther, false
	}
	return nil, true
}

//	Get first item from the queue without removing it
func (q *Queue) FetchHead(item interface{}) error {
	var items []interface{}

	entry, _, err := q.Client.KV().Get(q.Key, nil)
	if err != nil {
		return err
	}
	if entry == nil {
		return nil
	}

	if len(entry.Value) > 0 {
		if err := json.Unmarshal(entry.Value, &items); err != nil {
			return err
		}
	}

	if len(items) == 0 {
		return nil
	}

	d, err := json.Marshal(items[0])
	if err != nil {
		return err
	}

	if err := json.Unmarshal(d, &item); err != nil {
		return err
	}
	return nil
}

func (q *Queue) Items(items interface{}) error {
	entry, _, err := q.Client.KV().Get(q.Key, nil)
	if err != nil {
		return err
	}
	if entry == nil || len(entry.Value) == 0 {
		clearSlice(items)
		return nil
	}

	return json.Unmarshal(entry.Value, items)
}

func clearSlice(items interface{}) {
	if reflect.TypeOf(items).Kind() != reflect.Ptr {
		return
	}
	t := reflect.TypeOf(items).Elem()
	v := reflect.ValueOf(items).Elem()
	v.Set(reflect.MakeSlice(t, 0, 0))
}

func (q *Queue) Clear() error {
	_, err := q.Client.KV().Delete(q.Key, &api.WriteOptions{})
	return err
}

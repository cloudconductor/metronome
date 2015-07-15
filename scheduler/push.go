package scheduler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"scheduler/queue"
	"scheduler/util"

	"github.com/hashicorp/consul/api"
)

func Push() (string, error) {
	l, err := util.Consul().LockKey(LOCK_KEY)
	_, err = l.Lock(nil)
	if err != nil {
		return "", err
	}
	defer l.Unlock()

	eq := &queue.Queue{Client: util.Consul(), Key: EVENT_QUEUE_KEY}

	bytes, err := ioutil.ReadAll(os.Stdin)
	var receiveEvents []api.UserEvent
	err = json.Unmarshal(bytes, &receiveEvents)

	for _, re := range receiveEvents {
		err = pushSingleEvent(eq, re)
		if err != nil {
			return "", err
		}
	}
	return "", nil
}

func pushSingleEvent(eq *queue.Queue, re api.UserEvent) error {
	var storedEvents []api.UserEvent
	err := eq.Items(&storedEvents)
	if err != nil {
		return err
	}

	for _, se := range storedEvents {
		if se.ID == re.ID {
			fmt.Printf("Receive event was already registerd in a queue(ID: %s)\n", re.ID)
			return nil
		}
	}

	err = eq.EnQueue(re)
	if err != nil {
		return err
	}

	fmt.Printf("Push event to queue(ID: %s, Name: %s)\n", re.ID, re.Name)
	return nil
}

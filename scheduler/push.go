package scheduler

import (
	"encoding/json"
	"io/ioutil"
	"metronome/queue"
	"metronome/util"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

//	Push event to event queue when execute metronome from consul
func Push() (string, error) {
	l, err := util.Consul().LockKey(LOCK_KEY)
	if err != nil {
		return "", err
	}
	if _, err := l.Lock(nil); err != nil {
		return "", err
	}
	defer l.Unlock()

	//	Unmarshal STDIN from consul
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return "Following error has occurred while read STDIN", err
	}
	var receiveEvents []api.UserEvent
	err = json.Unmarshal(bytes, &receiveEvents)
	if err != nil {
		return "Following error has occurred while unmarshal STDIN", err
	}

	//	Enqueue each event to event queue
	eq := &queue.Queue{
		Client: util.Consul(),
		Key:    EVENT_QUEUE_KEY,
	}
	for _, re := range receiveEvents {
		if err := pushSingleEvent(eq, re); err != nil {
			return "", err
		}
	}
	return "", nil
}

func pushSingleEvent(eq *queue.Queue, re api.UserEvent) error {
	//	Reject received event if it had occurred already
	var storedEvents []api.UserEvent
	if err := eq.Items(&storedEvents); err != nil {
		return err
	}
	for _, se := range storedEvents {
		if se.ID == re.ID {
			log.Infof("Receive event was already registerd in a queue(ID: %s, Name: %s)", re.ID, re.Name)
			return nil
		}
	}

	//	Enqueue received event to event queue on consul
	if err := eq.EnQueue(re); err != nil {
		return err
	}

	log.Infof("Push event to queue(ID: %s, Name: %s)", re.ID, re.Name)
	return nil
}

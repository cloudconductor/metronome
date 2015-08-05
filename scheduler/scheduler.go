package scheduler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"metronome/config"
	"metronome/util"
	"path/filepath"
	"sort"

	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
)

const EVENT_QUEUE_KEY = "metronome/event_queue"
const PROGRESS_QUEUE_KEY = "metronome/progress_task_queue"
const LOCK_KEY = "metronome/event_queue/lock"

type Scheduler struct {
	schedules map[string]Schedule
	node      string
}

func NewScheduler() (*Scheduler, error) {
	scheduler := &Scheduler{}
	scheduler.schedules = make(map[string]Schedule)

	if err := scheduler.load(); err != nil {
		return nil, err
	}

	log.Info("Scheduler initialized")
	return scheduler, nil
}

func (scheduler *Scheduler) load() error {
	entries, err := ioutil.ReadDir(filepath.Join(config.BaseDir, "patterns"))
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		path := filepath.Join(config.BaseDir, "patterns", e.Name(), "task.yml")
		if !util.Exists(path) {
			log.Warnf("Schedule file does not found(%s)", path)
			continue
		}

		d, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to load config file(%s)\n\t%s", path, err))
		}
		var schedule Schedule
		if err := yaml.Unmarshal([]byte(d), &schedule); err != nil {
			return errors.New(fmt.Sprintf("Failed to unmarshal json(%s)\n\t%s", path, err))
		}

		schedule.PostUnmarshal(e.Name())
		scheduler.schedules[e.Name()] = schedule
		log.Debug(&schedule)
	}
	return nil
}

func (scheduler *Scheduler) sortedEvents(name string) Events {
	var events Events
	for _, v := range scheduler.schedules {
		e, found := v.Events[name]
		if !found {
			continue
		}
		events = append(events, *e)
	}
	sort.Sort(events)
	return events
}

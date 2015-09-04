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

//	Load shedule information from all task.yml
func (scheduler *Scheduler) load() error {
	for _, path := range config.Files {
		if path == "" {
			continue
		}
		if !util.Exists(path) {
			log.Warnf("Schedule file does not found(%s)", path)
			continue
		}
		log.Info(fmt.Sprintf("Load %s", path))

		d, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to load config file(%s)\n\t%s", path, err))
		}
		var schedule Schedule
		schedule.Default = taskDefault()
		if err := yaml.Unmarshal([]byte(d), &schedule); err != nil {
			return errors.New(fmt.Sprintf("Failed to unmarshal json(%s)\n\t%s", path, err))
		}

		patternName := patternName(path)
		schedule.PostUnmarshal(path, patternName)
		scheduler.schedules[patternName] = schedule
		log.Debug(&schedule)
	}
	return nil
}

//	Sort event by priority over all patterns
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

func taskDefault() map[string]interface{} {
	return map[string]interface{}{
		"timeout": float64(1800),
	}
}

func patternName(path string) string {
	return filepath.Base(filepath.Dir(path))
}

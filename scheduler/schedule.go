package scheduler

import (
	"fmt"
	"metronome/config"
	"metronome/task"
	"metronome/util"
	"os"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const INDENT_WIDTH = 2

//	Represent task.yml as golang structure
type Schedule struct {
	path         string
	pattern      string
	Environments map[string]string
	Variables    map[string]string
	Default      TaskDefault
	Events       map[string]*Event
	Tasks        map[string]*task.Task
}

//	Represent default node in task.yml
type TaskDefault struct {
	Timeout int32
}

//	Set path of task.yml and pattern directory to all tasks and operations
func (s *Schedule) PostUnmarshal(path string, pattern string) {
	s.path = path
	s.pattern = pattern
	for k, e := range s.Events {
		e.Name = k
		e.SetPattern(path, pattern)
	}
	for k, t := range s.Tasks {
		t.Name = k
		if t.Timeout == 0 {
			t.Timeout = s.Default.Timeout
		}

		t.SetPattern(path, pattern)
	}

	if s.Variables == nil {
		s.Variables = make(map[string]string)
	}
	s.Variables["role"] = config.Role

	s.setEnvironmentVariables()
}

func (s *Schedule) String() string {
	str := ""

	str += "Variables:\n"
	var variables []string
	for k, v := range s.Variables {
		variables = append(variables, fmt.Sprintf("%s: %s", k, v))
	}
	str += indent(strings.Join(variables, "\n"), 1) + "\n"
	str += "\n"

	str += "Default:\n"
	str += indent(fmt.Sprintf("%v", s.Default), 1) + "\n"
	str += "\n"

	str += "Events:\n"
	for k, v := range s.Events {
		str += indent(fmt.Sprintf("%s:", k), 1) + "\n"
		str += indent(fmt.Sprintf("%v", v), 2) + "\n"
	}
	str += "\n"

	str += "Tasks:\n"
	for k, v := range s.Tasks {
		str += indent(fmt.Sprintf("%s:", k), 1) + "\n"
		str += indent(fmt.Sprintf("%v", v), 2) + "\n"
	}
	return str
}

func indent(s string, n int) string {
	var results []string
	for _, r := range strings.Split(s, "\n") {
		for i := 0; i < n*INDENT_WIDTH; i++ {
			r = " " + r
		}
		results = append(results, r)
	}

	return strings.Join(results, "\n")
}

//	Set environment variables from environment node in task.yml
func (s *Schedule) setEnvironmentVariables() {
	r := regexp.MustCompile(`\$[a-zA-Z0-9_-]+`)
	for k, v := range s.Environments {
		v = r.ReplaceAllStringFunc(v, func(s string) string {
			return os.Getenv(s[1:len(s)])
		})
		v = util.ParseString(v, s.Variables)
		os.Setenv(k, v)
		log.Info(fmt.Sprintf("Set environment(%s): %s", k, v))
	}
}

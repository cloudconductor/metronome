package scheduler

import (
	"fmt"
	"scheduler/task"
	"strings"
)

const INDENT_WIDTH = 2

type Schedule struct {
	pattern   string
	Variables map[string]string
	Default   TaskDefault
	Events    map[string]Event
	Tasks     map[string]task.Task
}

type TaskDefault struct {
	Timeout    int32
	ChefConfig string `json:"chef_config"`
}

func (s *Schedule) SetPattern(pattern string) {
	s.pattern = pattern
	for _, t := range s.Tasks {
		t.SetPattern(pattern)
	}
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
		str += indent(fmt.Sprintf("%v", &v), 2) + "\n"
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

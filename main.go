package main

import (
	"fmt"
	"log"
	"os"
)

var stdlog, errlog *log.Logger

func init() {
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func main() {
	service, err := CreateService("scheduler", "Scheduler for consul event")
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}

	status, err := service.Manage()
	if err != nil {
		errlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}

	fmt.Println(status)
}

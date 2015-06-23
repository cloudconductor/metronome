package config

import (
	"flag"

	"github.com/monochromegane/conflag"
)

var (
	Node string

	Token              string
	Hostname           string
	Port               int
	Protocol           string
	InsecureSkipVerify bool

	ScheduleFile string

	ServiceManager string

	BaseDir string
)

func init() {
	flag.StringVar(&Node, "node", "", "Node name of this server on consul(default: Retrieve node from consul catalog)")

	flag.StringVar(&Token, "token", "", "Consul ACL token")
	flag.StringVar(&Hostname, "host", "127.0.0.1", "Consul host")
	flag.IntVar(&Port, "port", 8500, "Consul port")
	flag.StringVar(&Protocol, "protocol", "https", "Consul protocol (http / https)")
	flag.BoolVar(&InsecureSkipVerify, "insecure-skip-verify", false, "Skip server verification on SSL/TLS")

	flag.StringVar(&ScheduleFile, "schedule-file", "task.yml", "Load schedule from this file")

	flag.StringVar(&ServiceManager, "service-manager", "init", "Service manager(systemd / init)")

	flag.StringVar(&BaseDir, "base-dir", "/opt/cloudconductor", "CloudConductor base dir(default: /opt/cloudconductor))")

	if args, err := conflag.ArgsFrom("/etc/scheduler/config.yml"); err == nil {
		flag.CommandLine.Parse(args)
	}

	flag.Parse()
}

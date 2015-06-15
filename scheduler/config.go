package scheduler

import "flag"

type Config struct {
	Hostname           string
	Port               int
	Token              string
	Protocol           string
	InsecureSkipVerify bool
}

func (c *Config) Load() {
	flag.StringVar(&c.Hostname, "host", "127.0.0.1", "Consul host")
	flag.IntVar(&c.Port, "port", 8500, "Consul port")
	flag.StringVar(&c.Protocol, "protocol", "https", "Consul protocol (http / https)")
	flag.StringVar(&c.Token, "token", "", "Consul ACL token")
	flag.BoolVar(&c.InsecureSkipVerify, "insecure-skip-verify", false, "Skip server verification on SSL/TLS")
	flag.Parse()
}

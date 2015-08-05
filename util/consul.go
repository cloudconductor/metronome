package util

import (
	"crypto/tls"
	"fmt"
	"metronome/config"
	"net/http"

	"github.com/hashicorp/consul/api"
)

var consul *api.Client

func Consul() *api.Client {
	if consul == nil {
		c := api.DefaultConfig()
		c.Token = config.Token
		c.Address = fmt.Sprintf("%s:%d", config.Hostname, config.Port)
		c.Scheme = config.Protocol
		c.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.InsecureSkipVerify,
			},
		}

		var err error
		consul, err = api.NewClient(c)
		if err != nil {
			panic("Failed to create consul.Client")
		}
	}

	return consul
}

func HasCatalogRecord(node string, service string, tag string) bool {
	c, _, err := Consul().Catalog().Node(node, &api.QueryOptions{})
	if err != nil || c == nil {
		return false
	}

	if service == "" {
		return true
	}

	s, ok := c.Services[service]
	if !ok {
		return false
	}

	if tag == "" {
		return true
	}

	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}

	return false
}

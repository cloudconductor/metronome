package util

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"scheduler/config"

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

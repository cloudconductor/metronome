package config

import (
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/monochromegane/conflag"
)

const CONF_PATH string = "/etc/metronome/config.yml"
const VARIABLES_PATH string = "/etc/metronome/variables.yml"

var (
	UserVariables stringMapValue

	Token              string
	Hostname           string
	Port               int
	Protocol           string
	InsecureSkipVerify bool

	ProxyHost string
	ProxyPort int
	NoProxy   string

	ServiceManager string
	Shell          string

	Files []string

	Role string

	Debug bool
)

func init() {
	var files string

	UserVariables = loadUserVariables(VARIABLES_PATH)
	flag.Var(&UserVariables, "var", "Specify user variables(ex. \"-var key1=value1 -var key2=value2\")")

	flag.StringVar(&Token, "token", "", "Consul ACL token")
	flag.StringVar(&Hostname, "host", "127.0.0.1", "Consul host")
	flag.IntVar(&Port, "port", 8500, "Consul port")
	flag.StringVar(&Protocol, "protocol", "https", "Consul protocol (http / https)")
	flag.BoolVar(&InsecureSkipVerify, "insecure-skip-verify", false, "Skip server verification on SSL/TLS")

	flag.StringVar(&ProxyHost, "proxy-host", "", "Hostname or IP Address of proxy server")
	flag.IntVar(&ProxyPort, "proxy-port", 8080, "Port number of proxy server")
	flag.StringVar(&NoProxy, "no-proxy", "", "Hostname list without proxy server")

	flag.StringVar(&Shell, "shell", "/bin/sh", "Shell path(default: /bin/sh)")
	flag.StringVar(&ServiceManager, "service-manager", "init", "Service manager(systemd / init)")

	flag.StringVar(&files, "files", "", "Path list of task.yml")

	flag.StringVar(&Role, "role", "", "Role names of self instance(ex. \"-role web, ap\")")

	flag.BoolVar(&Debug, "debug", false, "Debug mode enabled(default: false)")

	if args, err := conflag.ArgsFrom(CONF_PATH); err == nil {
		flag.CommandLine.Parse(args)
	}

	flag.Parse()

	Files = strings.Split(files, ",")

	setEnvironmentVariables()
}

func loadUserVariables(path string) map[string]string {
	vars := make(map[string]string)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}

	if err := yaml.Unmarshal(b, &vars); err != nil {
		return nil
	}

	return vars
}

type stringMapValue map[string]string

func (v *stringMapValue) String() string {
	return ""
}

func (v *stringMapValue) Set(s string) error {
	if *v == nil {
		*v = make(map[string]string)
	}
	items := strings.Split(s, "=")
	(*v)[items[0]] = items[1]
	return nil
}

func setEnvironmentVariables() {
	if ProxyHost != "" {
		proxy := "http://" + ProxyHost + ":" + strconv.Itoa(ProxyPort)
		os.Setenv("http_proxy", proxy)
		os.Setenv("https_proxy", proxy)
		os.Setenv("ftp_proxy", proxy)
	}

	if NoProxy != "" {
		os.Setenv("no_proxy", NoProxy)
	}

	if Role != "" {
		os.Setenv("ROLE", Role)
	}
}

func GetValue(name string) string {
	switch name {
	case "token":
		return Token
	}
	return ""
}

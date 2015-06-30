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

const CONF_PATH string = "/etc/scheduler/config.yml"
const VARIABLES_PATH string = "/etc/scheduler/variables.yml"

var (
	UserVariables stringMapValue

	Token              string
	Hostname           string
	Port               int
	Protocol           string
	InsecureSkipVerify bool

	ProxyHost string
	ProxyPort int
	Path      string

	ServiceManager string

	BaseDir string

	Role string
)

func init() {
	UserVariables = loadUserVariables(VARIABLES_PATH)
	flag.Var(&UserVariables, "var", "Specify user variables(ex. \"-var key1=value1 -var key2=value2\")")

	flag.StringVar(&Token, "token", "", "Consul ACL token")
	flag.StringVar(&Hostname, "host", "127.0.0.1", "Consul host")
	flag.IntVar(&Port, "port", 8500, "Consul port")
	flag.StringVar(&Protocol, "protocol", "https", "Consul protocol (http / https)")
	flag.BoolVar(&InsecureSkipVerify, "insecure-skip-verify", false, "Skip server verification on SSL/TLS")

	flag.StringVar(&ProxyHost, "proxy-host", "", "Hostname or IP Address of proxy server")
	flag.IntVar(&ProxyPort, "proxy-port", 8080, "Port number of proxy server")
	flag.StringVar(&Path, "path", "", "Add PATH on environment variables")

	flag.StringVar(&ServiceManager, "service-manager", "init", "Service manager(systemd / init)")

	flag.StringVar(&BaseDir, "base-dir", "/opt/cloudconductor", "CloudConductor base dir(default: /opt/cloudconductor))")

	flag.StringVar(&Role, "role", "", "Role names of self instance(ex. \"-role web, ap\")")

	if args, err := conflag.ArgsFrom(CONF_PATH); err == nil {
		flag.CommandLine.Parse(args)
	}

	flag.Parse()

	setEnvironmentVariables()
}

func loadUserVariables(path string) map[string]string {
	vars := make(map[string]string)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}

	err = yaml.Unmarshal(b, &vars)
	if err != nil {
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

	if Path != "" {
		os.Setenv("PATH", Path+":"+os.Getenv("PATH"))
	}
}

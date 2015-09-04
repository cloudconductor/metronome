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
	//	user specified variables that can reference from task.yml with {{XXX}} format
	UserVariables stringMapValue

	//	Connection parameter for Consul server
	Token              string
	Hostname           string
	Port               int
	Protocol           string
	InsecureSkipVerify bool

	//	HTTP/HTTPS proxy settings
	ProxyHost string
	ProxyPort int
	NoProxy   string

	//	Type of service manager(systemd / init)
	ServiceManager string

	//	All paths of task.yml
	Files []string

	//	Instance role
	Role string

	//	Skip event that doesn't execute on any instance
	Skippable bool

	//	Enable debug output and features
	Debug bool
)

func init() {
	var files string

	//	load user variables from file
	UserVariables = loadUserVariables(VARIABLES_PATH)
	flag.Var(&UserVariables, "var", "Specify user variables(ex. \"-var key1=value1 -var key2=value2\")")

	//	load options from commandline parameter or file
	flag.StringVar(&Token, "token", "", "Consul ACL token")
	flag.StringVar(&Hostname, "host", "127.0.0.1", "Consul host")
	flag.IntVar(&Port, "port", 8500, "Consul port")
	flag.StringVar(&Protocol, "protocol", "https", "Consul protocol (http / https)")
	flag.BoolVar(&InsecureSkipVerify, "insecure-skip-verify", false, "Skip server verification on SSL/TLS")

	flag.StringVar(&ProxyHost, "proxy-host", "", "Hostname or IP Address of proxy server")
	flag.IntVar(&ProxyPort, "proxy-port", 8080, "Port number of proxy server")
	flag.StringVar(&NoProxy, "no-proxy", "", "Hostname list without proxy server")

	flag.StringVar(&ServiceManager, "service-manager", "init", "Service manager(systemd / init)")

	flag.StringVar(&files, "files", "", "Path list of task.yml")

	flag.StringVar(&Role, "role", "", "Role names of self instance(ex. \"-role web, ap\")")

	flag.BoolVar(&Skippable, "skippable", true, "Skip task which isn't needed by anyone(default: true)")

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

//	Create map structure when load variables automatically
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

//	Set environment variables on this process
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
}

//	Return configuration for task.yml with {{config.XXXX}} format
func GetValue(name string) string {
	switch name {
	case "token":
		return Token
	case "host":
		return Hostname
	case "port":
		return strconv.Itoa(Port)
	case "protocol":
		return Protocol
	case "insecure-skip-verify":
		return strconv.FormatBool(InsecureSkipVerify)
	case "proxy-host":
		return ProxyHost
	case "proxy-port":
		return strconv.Itoa(ProxyPort)
	case "no-proxy":
		return NoProxy
	case "service-manager":
		return ServiceManager
	case "role":
		return Role
	case "skippable":
		return strconv.FormatBool(Skippable)
	case "debug":
		return strconv.FormatBool(Debug)
	}
	return ""
}

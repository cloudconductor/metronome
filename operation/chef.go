package operation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"metronome/config"
	"metronome/util"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
	"github.com/imdario/mergo"
)

const BERKS_VENDOR_ERROR = 139

//	Execute chef-solo with specified parameter
type ChefOperation struct {
	BaseOperation
	RunList        []string `json:"run_list"`
	Configurations map[string]interface{}
	Attributes     map[string]interface{}
}

func NewChefOperation(v json.RawMessage) *ChefOperation {
	o := &ChefOperation{}
	json.Unmarshal(v, &o)

	return o
}

func (o *ChefOperation) Run(vars map[string]string) error {
	//	Filter runlist by JSON file existance in roles directory
	runlist := o.ensureRunList(o.parseRunList(o.RunList, vars))

	//	Create attributes JSON for chef-solo
	json, err := o.createJson(runlist, util.ParseMap(o.Attributes, vars))
	if err != nil {
		return err
	}

	//	Create configuration file for chef-solo
	conf, err := o.createConf(vars)
	if err != nil {
		return err
	}

	//	Execute berkshelf to get depencency cookbooks
	if err := o.executeBerkshelf(); err != nil {
		return err
	}

	//	Execute chef-solo with configuration file and attribute JSON
	return o.executeChef(conf, json)
}

//	Convert {{role}} in task.yml to array of individual role with 'all' role
//	When role is 'web,ap', convert from 'role[{{role}}_deploy]' to role[all_deploy], role[web_deploy] and role[ap_deploy]
func (o *ChefOperation) parseRunList(runlist []string, vars map[string]string) []string {
	var results []string
	for _, v := range runlist {
		if strings.Contains(v, "{{role}}") {
			roles := append([]string{"all"}, strings.Split(config.Role, ",")...)
			for _, role := range roles {
				results = append(results, strings.Replace(v, "{{role}}", role, -1))
			}
		} else {
			results = append(results, v)
		}
	}
	return util.ParseArray(results, vars)
}

//	Filter runlist by JSON file existance in roles directory
func (o *ChefOperation) ensureRunList(runlist []string) []string {
	var results []string
	r, _ := regexp.Compile("^role\\[(.*)\\]$")
	for _, v := range runlist {
		matches := r.FindStringSubmatch(v)
		if len(matches) > 0 {
			if !util.Exists(filepath.Join(o.patternDir(), "roles", matches[1]+".json")) {
				continue
			}
		}
		results = append(results, v)
	}
	return results
}

func (o *ChefOperation) createJson(runlist []string, overwriteAttributes map[string]interface{}) (string, error) {
	var err error

	//	Get cloudconductor/parameters from consul KVS and overwrite some attributes by specified parameter in task.yml
	cloudconductor, err := getAttributes(overwriteAttributes)
	if err != nil {
		return "", err
	}

	//	Get cloudconductor/servers from consul KVS
	servers, err := getServers()
	if err != nil {
		servers = make(map[string]interface{})
	}

	//	cloudconductor/patterns/
	attributes, err := extractAttributes(cloudconductor)
	if err != nil {
		return "", err
	}

	json, err := writeJson(runlist, cloudconductor, servers, attributes)
	if err != nil {
		return "", err
	}
	return json, nil
}

func getAttributes(overwriteAttributes map[string]interface{}) (map[string]interface{}, error) {
	//	Get cloudconductor/parameters from consul KVS
	var attributes map[string]interface{}
	var c *api.Client = util.Consul()
	kv, _, err := c.KV().Get("cloudconductor/parameters", &api.QueryOptions{})
	if err == nil && kv != nil {
		if err := json.Unmarshal(kv.Value, &attributes); err != nil {
			return nil, err
		}
	} else {
		attributes = make(map[string]interface{})
		attributes["cloudconductor"] = make(map[string]interface{})
		attributes["cloudconductor"].(map[string]interface{})["patterns"] = make(map[string]interface{})
	}

	//	Overwrite some attributes by specified parameter in task.yml
	if err := mergeAttributes(attributes, overwriteAttributes); err != nil {
		return nil, err
	}
	return attributes, nil
}

func mergeAttributes(src, dst map[string]interface{}) error {
	patterns := src["cloudconductor"].(map[string]interface{})["patterns"].(map[string]interface{})

	for k, v := range dst {
		//	Overwrite each pattern JSON by specified map
		if patterns[k] == nil {
			pattern := make(map[string]interface{})
			pattern["user_attributes"] = make(map[string]interface{})
			patterns[k] = pattern
		}
		m := patterns[k].(map[string]interface{})["user_attributes"].(map[string]interface{})
		if err := mergo.MergeWithOverwrite(&m, v); err != nil {
			return errors.New(fmt.Sprintf("Failed to merge attributes(%v)", err))
		}
	}
	return nil
}

//	Aggregate cloudconductor/servers/* and return it to output to attribute JSON
func getServers() (map[string]interface{}, error) {
	var c *api.Client = util.Consul()
	consulServers, _, err := c.KV().List("cloudconductor/servers", &api.QueryOptions{})
	if err != nil {
		return nil, err
	}
	servers := make(map[string]interface{})
	for _, s := range consulServers {
		node := strings.TrimPrefix(s.Key, "cloudconductor/servers/")
		v := make(map[string]interface{})
		if err := json.Unmarshal(s.Value, &v); err != nil {
			return nil, err
		}
		servers[node] = v
	}
	return servers, nil
}

//	Extract attributes to support node['pattern_name']['XXXX'] in chef recipes
func extractAttributes(src map[string]interface{}) (map[string]interface{}, error) {
	var results map[string]interface{}

	patterns := src["cloudconductor"].(map[string]interface{})["patterns"].(map[string]interface{})
	for _, v := range patterns {
		m := v.(map[string]interface{})["user_attributes"].(map[string]interface{})
		if err := mergo.MergeWithOverwrite(&results, m); err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to merge attributes(%v)", err))
		}
	}
	return results, nil
}

func writeJson(runlist []string, cloudconductor map[string]interface{}, servers map[string]interface{}, attributes map[string]interface{}) (string, error) {
	//	Construct attribute json structure
	m := make(map[string]interface{})
	m["run_list"] = runlist
	m["cloudconductor"] = cloudconductor["cloudconductor"]
	m["cloudconductor"].(map[string]interface{})["servers"] = servers
	mergo.MergeWithOverwrite(&m, attributes)
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	//	Output attribute JSON to temporary file
	f, err := ioutil.TempFile("", "chef-node-json")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		return "", err
	}

	return f.Name(), nil
}

func (o *ChefOperation) createConf(vars map[string]string) (string, error) {
	//	Overwrite configuration by user specified configuration in task.yml
	m, err := o.defaultConfig()
	if err != nil {
		return "", err
	}
	if err := mergo.MergeWithOverwrite(&m, o.Configurations); err != nil {
		return "", err
	}

	//	Output configuration to temporary file
	f, err := ioutil.TempFile("", "chef-conf")
	if err != nil {
		return "", err
	}
	defer f.Close()
	for k, v := range m {
		if _, err := f.WriteString(fmt.Sprintf("%s %s\n", k, convertRubyCode(v))); err != nil {
			return "", err
		}
	}

	return f.Name(), nil
}

func convertRubyCode(v interface{}) string {
	//	Convert to appropriate format as ruby for chef configuration file
	switch v.(type) {
	case string:
		if strings.HasPrefix(v.(string), ":") {
			return v.(string)
		} else {
			return "'" + v.(string) + "'"
		}
	case []string:
		var values []string
		for _, e := range v.([]string) {
			values = append(values, "'"+e+"'")
		}
		return "[" + strings.Join(values, ",") + "]"
	}

	return ""
}

//	Return default configuration for chef-solo
func (o *ChefOperation) defaultConfig() (map[string]interface{}, error) {
	m := map[string]interface{}{
		"ssl_verify_mode": ":verify_peer",
		"role_path":       []string{},
		"log_level":       ":info",
		"log_location":    "",
		"file_cache_path": "",
		"cookbook_path":   []string{},
	}

	var roleDirs []string
	var cookbookDirs []string

	roleDirs = []string{filepath.Join(o.patternDir(), "roles")}
	cookbookDirs = []string{filepath.Join(o.patternDir(), "cookbooks"), filepath.Join(o.patternDir(), "site-cookbooks")}

	m["log_location"] = filepath.Join(o.patternDir(), "logs", o.pattern+"_chef-solo.log")
	m["file_cache_path"] = filepath.Join(o.patternDir(), "tmp", "cache")
	m["role_path"] = roleDirs
	m["cookbook_path"] = cookbookDirs

	return m, nil
}

func (o *ChefOperation) executeBerkshelf() error {
	//	Check Berksfile in target pattern
	if !util.Exists(filepath.Join(o.patternDir(), "Berksfile")) {
		log.Debug("chef: Skip berkshelf because Berksfile doesn't found in pattern directory")
		return nil
	}

	//	Execute berkshelf and ignore specified error
	log.Info("chef: Execute berkshelf")
	cmd := exec.Command("berks", "vendor", "cookbooks")
	cmd.Dir = o.patternDir()
	env := os.Environ()
	env = append(env, "HOME=/root")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	log.Debug(string(out))

	if err != nil {
		if e2, ok := err.(*exec.ExitError); ok {
			if s, ok := e2.Sys().(syscall.WaitStatus); ok {
				if s.ExitStatus() == BERKS_VENDOR_ERROR {
					return nil
				}
			}
		}
	}
	return err
}

func (o *ChefOperation) executeChef(conf string, json string) error {
	//	Delete temporary files automatically without debug mode
	if !config.Debug {
		defer os.Remove(conf)
		defer os.Remove(json)
	}

	//	Execute chef with attribute JSON and configuration file
	log.Infof("chef: Execute chef(conf: %s, json: %s)", conf, json)
	cmd := exec.Command("chef-solo", "-c", conf, "-j", json)
	cmd.Dir = o.patternDir()
	env := os.Environ()
	env = append(env, "HOME=/root")
	env = append(env, "CONSUL_SECRET_KEY="+config.Token)
	env = append(env, "ROLE="+config.Role)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	log.Debug(string(out))
	return err
}

func (o *ChefOperation) patternDir() string {
	return filepath.Dir(o.path)
}

func (o *ChefOperation) String() string {
	return "chef"
}

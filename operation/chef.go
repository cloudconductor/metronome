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
	AttributeKeys  []string `json:"attribute_keys"`
}

func NewChefOperation(v json.RawMessage) *ChefOperation {
	o := &ChefOperation{}
	json.Unmarshal(v, &o)

	return o
}

func (o *ChefOperation) SetDefault(m map[string]interface{}) {
	if len(o.AttributeKeys) == 0 {
		if v, ok := m["attribute_keys"]; ok {
			if keys, ok := v.([]interface{}); ok {
				for _, key := range keys {
					o.AttributeKeys = append(o.AttributeKeys, key.(string))
				}
			}
		}
	}
}

func (o *ChefOperation) Run(vars map[string]string) error {
	//	Filter runlist by JSON file existance in roles directory
	runlist := o.ensureRunList(o.parseRunList(o.RunList, vars))

	//	Create attributes JSON for chef-solo
	json, err := o.createJson(runlist, util.ParseArray(o.AttributeKeys, vars), util.ParseMap(o.Attributes, vars))
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

func (o *ChefOperation) createJson(runlist []string, keys []string, overwriteAttributes map[string]interface{}) (string, error) {
	var err error

	//	Get attribute json from consul KVS and overwrite some attributes by specified parameter in task.yml
	attributes, err := getAttributes(keys, overwriteAttributes)
	if err != nil {
		//	Execute chef without attributes when consul hasn't started
		attributes = make(map[string]interface{})
	}

	json, err := writeJson(runlist, attributes)
	if err != nil {
		return "", err
	}
	return json, nil
}

func getAttributes(keys []string, overwriteAttributes map[string]interface{}) (map[string]interface{}, error) {
	var attributes map[string]interface{}
	var c *api.Client = util.Consul()
	attributes = make(map[string]interface{})

	//	Get attributes from consul KVS
	for _, key := range keys {
		list, _, err := c.KV().List(key, &api.QueryOptions{})
		if err != nil {
			return nil, err
		}

		for _, kv := range list {
			var a map[string]interface{}
			if err := json.Unmarshal(kv.Value, &a); err != nil {
				return nil, err
			}
			if err := mergo.MergeWithOverwrite(&attributes, a); err != nil {
				return nil, errors.New(fmt.Sprintf("Failed to merge attributes(%v)", err))
			}
		}
	}

	//	Overwrite some attributes by specified parameter in task.yml
	if err := mergo.MergeWithOverwrite(&attributes, overwriteAttributes); err != nil {
		return nil, err
	}
	return attributes, nil
}

func writeJson(runlist []string, attributes map[string]interface{}) (string, error) {
	//	Construct attribute json structure
	attributes["run_list"] = runlist
	b, err := json.Marshal(attributes)
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
	if err != nil {
		log.Error("Chef STDOUT")
		log.Error(string(out))
	} else {
		log.Debug("Chef STDOUT")
		log.Debug(string(out))
	}
	return err
}

func (o *ChefOperation) patternDir() string {
	return filepath.Dir(o.path)
}

func (o *ChefOperation) String() string {
	return "chef"
}

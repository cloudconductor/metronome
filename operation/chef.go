package operation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"scheduler/config"
	"scheduler/util"
	"strings"
	"syscall"

	"github.com/hashicorp/consul/api"
	"github.com/imdario/mergo"
)

const BERKS_VENDOR_ERROR = 139

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
	runlist := o.ensureRunList(o.parseRunList(o.RunList, vars))

	json, err := o.createJson(runlist, util.ParseMap(o.Attributes, vars))
	if err != nil {
		return err
	}

	conf, err := o.createConf(vars)
	if err != nil {
		return err
	}

	err = o.executeBerkshelf()
	if err != nil {
		return err
	}

	return o.executeChef(conf, json)
}

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

	cloudconductor, err := getAttributes(overwriteAttributes)
	if err != nil {
		return "", err
	}

	servers, err := getServers()
	if err != nil {
		servers = make(map[string]interface{})
	}

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
	var attributes map[string]interface{}
	var c *api.Client = util.Consul()
	kv, _, err := c.KV().Get("cloudconductor/parameters", &api.QueryOptions{})
	if err == nil && kv != nil {
		err = json.Unmarshal(kv.Value, &attributes)
		if err != nil {
			return nil, err
		}
	} else {
		attributes = make(map[string]interface{})
		attributes["cloudconductor"] = make(map[string]interface{})
		attributes["cloudconductor"].(map[string]interface{})["patterns"] = make(map[string]interface{})
	}

	err = mergeAttributes(attributes, overwriteAttributes)
	if err != nil {
		return nil, err
	}
	return attributes, nil
}

func mergeAttributes(src, dst map[string]interface{}) error {
	patterns := src["cloudconductor"].(map[string]interface{})["patterns"].(map[string]interface{})

	for k, v := range dst {
		if patterns[k] == nil {
			pattern := make(map[string]interface{})
			pattern["user_attributes"] = make(map[string]interface{})
			patterns[k] = pattern
		}
		m := patterns[k].(map[string]interface{})["user_attributes"].(map[string]interface{})
		err := mergo.MergeWithOverwrite(&m, v)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to merge attributes(%v)", err))
		}
	}
	return nil
}

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
		err = json.Unmarshal(s.Value, &v)
		servers[node] = v
		if err != nil {
			return nil, err
		}
	}
	return servers, nil
}

func extractAttributes(src map[string]interface{}) (map[string]interface{}, error) {
	var results map[string]interface{}

	patterns := src["cloudconductor"].(map[string]interface{})["patterns"].(map[string]interface{})
	for _, v := range patterns {
		m := v.(map[string]interface{})["user_attributes"].(map[string]interface{})
		err := mergo.MergeWithOverwrite(&results, m)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to merge attributes(%v)", err))
		}
	}
	return results, nil
}

func writeJson(runlist []string, cloudconductor map[string]interface{}, servers map[string]interface{}, attributes map[string]interface{}) (string, error) {
	m := make(map[string]interface{})
	m["run_list"] = runlist
	m["cloudconductor"] = cloudconductor["cloudconductor"]
	m["cloudconductor"].(map[string]interface{})["servers"] = servers
	mergo.MergeWithOverwrite(&m, attributes)

	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	f, err := ioutil.TempFile("", "chef-node-json")
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.Write(b)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func (o *ChefOperation) createConf(vars map[string]string) (string, error) {
	f, err := ioutil.TempFile("", "chef-conf")
	if err != nil {
		return "", err
	}
	defer f.Close()

	m, err := o.defaultConfig()
	if err != nil {
		return "", err
	}

	err = mergo.MergeWithOverwrite(&m, o.Configurations)
	if err != nil {
		return "", err
	}

	for k, v := range m {
		_, err = f.WriteString(fmt.Sprintf("%s %s\n", k, convertRubyCode(v)))
		if err != nil {
			return "", err
		}
	}

	return f.Name(), nil
}

func convertRubyCode(v interface{}) string {
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
	cmd := exec.Command("berks", "vendor", "cookbooks")
	cmd.Dir = o.patternDir()
	env := os.Environ()
	env = append(env, "HOME=/root")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))

	if err != nil {
		if e2, ok := err.(*exec.ExitError); ok {
			if s, ok := e2.Sys().(syscall.WaitStatus); ok {
				if s.ExitStatus() == BERKS_VENDOR_ERROR {
					return nil
				}
			}
		}
	}
	fmt.Println(err)
	return err
}

func (o *ChefOperation) executeChef(conf string, json string) error {
	defer os.Remove(conf)
	defer os.Remove(json)

	fmt.Printf("Execute chef(conf: %s, json: %s)\n", conf, json)
	cmd := exec.Command("chef-solo", "-c", conf, "-j", json)
	cmd.Dir = o.patternDir()
	env := os.Environ()
	env = append(env, "HOME=/root")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	fmt.Println(err)
	return err
}

func (o *ChefOperation) patternDir() string {
	return filepath.Join(config.BaseDir, "patterns", o.pattern)
}

func (o *ChefOperation) String() string {
	return "chef"
}

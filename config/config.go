
package config

import (
	"errors"
	"fmt"
	"github.com/HotelsDotCom/flyte-client/flyte"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"os"
)

type Pack struct {
	Id       string            `json:"id" yaml:"id"`
	Name     string            `json:"name" yaml:"name"`
	Envs     map[string]string `json:"envs" yaml:"envs"`
	Commands []Command         `json:"commands" yaml:"commands"`
}

type Command struct {
	Name    string            `json:"name" yaml:"name"`
	Input   map[string]string `json:"input" yaml:"input"`
	Request Request           `json:"request" yaml:"request"`
}

type Request struct {
	Type    string            `json:"type" yaml:"type"`
	Path    string            `json:"path" yaml:"path"`
	Auth    Auth              `json:"auth" yaml:"auth"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Data    string            `json:"data" yaml:"data"`
}

type Auth struct {
	User string `json:"user" yaml:"user"`
	Pass string `json:"pass" yaml:"pass"`
}

func NewPackDef(path string) (flyte.PackDef, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return flyte.PackDef{}, errors.New(fmt.Sprintf("could not read file %s, err: %s", path, err))
	}

	var cfg Pack
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return flyte.PackDef{}, errors.New(fmt.Sprintf("could not unmarshal file %s, err: %s", path, err))
	}

	envs, err := processEnvs(cfg)
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return flyte.PackDef{}, errors.New(fmt.Sprintf("could not process environment variables, err: %s", err))
	}

	commands, events, err := GenerateCommandsAndEvents(cfg, envs)
	if err != nil {
		return flyte.PackDef{}, errors.New(fmt.Sprintf("could not generate commands and/or events, err: %s", err))
	}

	return flyte.PackDef{Name: cfg.Name, Commands: commands, EventDefs: events}, nil
}

func processEnvs(cfg Pack) (map[string]string, error) {
	envs := make(map[string]string)
	exists := false
	for k, v := range cfg.Envs {
		if envs[v], exists = os.LookupEnv(k); !exists {
			return map[string]string{}, errors.New(fmt.Sprintf("%s environment variable is not set", k))
		}
	}
	return envs, nil
}

func (a Auth) Enabled() bool {
	return a.Pass != "" && a.User != ""
}

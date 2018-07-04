package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HotelsDotCom/flyte-client/flyte"
	"github.com/HotelsDotCom/go-logger"
	"net/http"
	"strings"
)

type command struct {
	envs         map[string]string
	cfg          Command
	failureEvent flyte.EventDef
	successEvent flyte.EventDef
}

func newHandlerAndEvents(cfg Command, envs map[string]string) (func(input json.RawMessage) flyte.Event, []flyte.EventDef, error) {
	cd := command{cfg: cfg, envs: envs}

	cd.successEvent = flyte.EventDef{Name: fmt.Sprintf("%sSuccess", cfg.Name)}
	cd.failureEvent = flyte.EventDef{Name: fmt.Sprintf("%sFailure", cfg.Name)}

	handler, err := cd.createHandler()
	if err != nil {
		return nil, []flyte.EventDef{}, err
	}
	return handler, []flyte.EventDef{cd.successEvent, cd.failureEvent}, nil
}

func (cd *command) createHandler() (func(input json.RawMessage) flyte.Event, error) {

	return func(input json.RawMessage) flyte.Event {

		resolvedInputs, err := cd.resolveInputs(input)
		if err != nil {
			logger.Error(err)
			return flyte.Event{EventDef: cd.failureEvent, Payload: err}
		}

		// Inject variable values in path and data
		path := injectVars(cd.cfg.Request.Path, resolvedInputs, cd.envs)

		r, err := cd.createRequest(resolvedInputs, path)
		if err != nil {
			logger.Error(err)
			return flyte.Event{EventDef: cd.failureEvent, Payload: err}
		}
		cd.setHeaders(r)

		payload, err := sendRequest(r)
		if err != nil {
			logger.Error(err)
			return flyte.Event{EventDef: cd.failureEvent, Payload: err}
		}

		logger.Debug(fmt.Sprintf("%s payload: %+v", cd.successEvent.Name, payload))
		return flyte.Event{EventDef: cd.successEvent, Payload: payload}
	}, nil
}

func (cd *command) createRequest(resolvedInputs map[string]string, path string) (*http.Request, error) {
	if cd.cfg.Request.Type == http.MethodPost || cd.cfg.Request.Type == http.MethodPut {
		data := injectVars(cd.cfg.Request.Data, resolvedInputs, cd.envs)
		r, err := http.NewRequest(cd.cfg.Request.Type, path, bytes.NewBuffer([]byte(data)))
		if err != nil {
			return nil, err
		}
		return r, nil
	}

	r, err := http.NewRequest(cd.cfg.Request.Type, path, nil)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (cd *command) resolveInputs(input json.RawMessage) (map[string]string, error) {
	var in map[string]string
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, err
	}

	// Resolve inputs
	resolvedInputs := make(map[string]string)
	for k, v := range cd.cfg.Input {
		resolvedInputs[v] = in[k]
	}

	return resolvedInputs, nil
}

func injectVars(s string, subs ...map[string]string) string {
	for _, sub := range subs {
		for k, v := range sub {
			s = strings.Replace(s, k, v, -1)
		}
	}
	return s
}

func (cd *command) setHeaders(r *http.Request) {
	for k, v := range cd.cfg.Request.Headers {
		r.Header.Set(k, v)
	}
	if cd.cfg.Request.Auth.Enabled() {
		r.SetBasicAuth(injectVars(cd.cfg.Request.Auth.User, cd.envs), injectVars(cd.cfg.Request.Auth.Pass, cd.envs))
	}
}

func sendRequest(r *http.Request) (interface{}, error) {
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		return err, nil
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return errors.New(fmt.Sprintf("bad response: code: %s", response.StatusCode)), nil
	}

	var responseBody interface{}
	return responseBody, json.NewDecoder(response.Body).Decode(&responseBody)
}

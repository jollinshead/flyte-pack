
package config

import (
	"encoding/json"
	"github.com/HotelsDotCom/flyte-client/flyte"
	"strings"
	"errors"
	"fmt"
	"github.com/HotelsDotCom/go-logger"
	"bytes"
	"net/http"
)

type command struct {
	envs map[string]string
	cfg  Command
	failureEvent flyte.EventDef
	successEvent flyte.EventDef
}

func newHandlerAndEvents(cfg Command, envs map[string]string) (func(input json.RawMessage) flyte.Event, []flyte.EventDef, error) {
	cd := command{cfg:cfg, envs:envs}

	cd.successEvent = flyte.EventDef{Name:fmt.Sprintf("%sSuccess", cfg.Name)}
	cd.failureEvent = flyte.EventDef{Name:fmt.Sprintf("%sFailure", cfg.Name)}

	switch strings.ToUpper(cd.cfg.Request.Type) {
	case http.MethodPost:
		handler, err := cd.createPostHandler()
		if err != nil {
			return nil, []flyte.EventDef{}, err
		}
		return handler, []flyte.EventDef{cd.successEvent, cd.failureEvent}, nil

	case http.MethodGet:
		handler, err := cd.createGetHandler()
		if err != nil {
			return nil, []flyte.EventDef{}, err
		}
		return handler, []flyte.EventDef{cd.successEvent, cd.failureEvent}, nil
	}

	return nil, []flyte.EventDef{}, errors.New(fmt.Sprintf("unknown request type '%s', ", cd.cfg.Request.Type))
}

func (cd *command) createGetHandler() (func(input json.RawMessage) flyte.Event, error) {

	return func(input json.RawMessage) flyte.Event {

		var in map[string]string
		if err := json.Unmarshal(input, &in); err != nil {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:err}
		}

		// Resolve inputs
		resolvedInputs := make(map[string]string)
		for k, v := range cd.cfg.Input {
			resolvedInputs[v] = in[k]
		}

		// Inject variable values in path and data
		path := injectVars(cd.cfg.Request.Path, resolvedInputs, cd.envs)

		req, err := cd.constructGetRequest(path)
		if err != nil {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:err}
		}

		var payload interface{}
		err = sendRequest(req, &payload)
		if err != nil {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:err}
		}

		return flyte.Event{EventDef:cd.successEvent, Payload:payload}
	}, nil
}

func (cd *command) constructGetRequest(path string) (*http.Request, error) {

	r, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return r, err
	}

	for k, v := range cd.cfg.Request.Headers {
		r.Header.Set(k, v)
	}

	if cd.cfg.Request.Auth.Enabled() {
		r.SetBasicAuth(injectVars(cd.cfg.Request.Auth.User,cd.envs), injectVars(cd.cfg.Request.Auth.Pass,cd.envs))
	}

	return r, nil
}

func (cd *command) createPostHandler() (func(input json.RawMessage) flyte.Event, error) {

	return func(input json.RawMessage) flyte.Event {

		var in map[string]string
		if err := json.Unmarshal(input, &in); err != nil {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:err}
		}

		// Resolve inputs
		resolvedInputs := make(map[string]string)
		for k, v := range cd.cfg.Input {
			resolvedInputs[v] = in[k]
		}

		// Inject variable values in path and data
		path := injectVars(cd.cfg.Request.Path, resolvedInputs, cd.envs)
		data := injectVars(cd.cfg.Request.Data, resolvedInputs, cd.envs)

		req, err := cd.constructPostRequest(path, data)
		if err != nil {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:err}
		}

		var payload interface{}
		err = sendRequest(req, &payload)
		if err != nil {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:err}
		}

		return flyte.Event{EventDef:cd.successEvent, Payload:payload}
	}, nil
}

func (cd *command) constructPostRequest(path, data string) (*http.Request, error) {

	//b, err := json.Marshal(data)
	//if err != nil {
	//	return nil, err
	//}

	r, err := http.NewRequest(http.MethodPost, path, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return r, err
	}

	for k, v := range cd.cfg.Request.Headers {
		r.Header.Set(k, v)
	}

	if cd.cfg.Request.Auth.Enabled() {
		r.SetBasicAuth(injectVars(cd.cfg.Request.Auth.User,cd.envs), injectVars(cd.cfg.Request.Auth.Pass,cd.envs))
	}

	return r, nil
}

func sendRequest(r *http.Request, responseBody interface{}) error {
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return errors.New(fmt.Sprintf("bad response: code: %s", response.StatusCode))
	}

	return json.NewDecoder(response.Body).Decode(&responseBody)
}

func injectVars(s string, subs ...map[string]string) string {
	for _, sub := range subs {
		for k, v := range sub {
			s = strings.Replace(s, k, v, -1)
		}
	}
	return s
}

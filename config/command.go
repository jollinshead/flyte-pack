
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
	}

	return nil, []flyte.EventDef{}, errors.New(fmt.Sprintf("unknown request type '%s', ", cd.cfg.Request.Type))
}

func (cd *command) createPostHandler() (func(input json.RawMessage) flyte.Event, error) {

	return func(input json.RawMessage) flyte.Event {

		var in interface{}
		if err := json.Unmarshal(input, &in); err != nil {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:err}
		}

		// Resolve inputs
		resolvedInputs := make(map[string]string)
		for k, v := range cd.cfg.Input {
			resolvedInputs[v] = in.(map[string]string)[k]
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
		res, err := sendRequest(req, &payload)
		if err != nil {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:err}
		}
		if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
			logger.Error(err)
			return flyte.Event{EventDef:cd.failureEvent, Payload:fmt.Sprintf("bad responce: code: %s, body: %+v", res.StatusCode, payload)}
		}

		return flyte.Event{EventDef:cd.successEvent, Payload:payload}
	}, nil
}

func (cd *command) constructPostRequest(path, data string) (*http.Request, error) {

	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(b))
	if err != nil {
		return r, err
	}

	for k, v := range cd.cfg.Request.Headers {
		r.Header.Set(k, v)
	}

	if cd.cfg.Request.Auth.Enabled() {
		r.SetBasicAuth(cd.cfg.Request.Auth.User, cd.cfg.Request.Auth.Pass)
	}

	return r, nil
}

func sendRequest(r *http.Request, responseBody interface{}) (*http.Response, error) {
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		return response, err
	}

	defer response.Body.Close()
	return response, json.NewDecoder(response.Body).Decode(&responseBody)
}

func injectVars(s string, subs ...map[string]string) string {
	for _, sub := range subs {
		for k, v := range sub {
			s = strings.Replace(s, k, v, -1)
		}
	}
	return s
}

package config

import (
	"errors"
	"fmt"
	"github.com/HotelsDotCom/flyte-client/flyte"
)

func GenerateCommandsAndEvents(cfg Pack, envs map[string]string) ([]flyte.Command, []flyte.EventDef, error) {
	var fcs []flyte.Command
	var fed []flyte.EventDef
	for _, v := range cfg.Commands {
		Handler, OutputEvents, err := newHandlerAndEvents(v, envs)
		if err != nil {
			return []flyte.Command{}, []flyte.EventDef{}, errors.New(fmt.Sprintf("could not create handler, err: %s", err))
		}

		fc := flyte.Command{Name: v.Name, OutputEvents: OutputEvents, Handler: Handler}
		fcs = append(fcs, fc)
		fed = append(fed, OutputEvents...)
	}

	return fcs, fed, nil
}

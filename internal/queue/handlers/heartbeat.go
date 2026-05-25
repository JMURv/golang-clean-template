package handlers

import (
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/nats-io/nats.go"
)

type Heartbeat struct {
	ctrl ctrl.AppCtrl
}

func (h Heartbeat) Handle(_ *nats.Msg) error {
	return nil
}

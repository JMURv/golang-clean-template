package handlers

import (
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/nats-io/nats.go"
)

type HandlerT interface {
	Handle(msg *nats.Msg) error
}

type Router struct {
	handlers map[string]HandlerT
}

func NewRouter(ctrl ctrl.AppCtrl) Router {
	router := Router{handlers: make(map[string]HandlerT)}

	router.RegisterHandler("heartbeat", Heartbeat{ctrl})
	return router
}

func (r *Router) RegisterHandler(subj string, handler HandlerT) {
	r.handlers[subj] = handler
}

func (r *Router) Get(subj string) HandlerT {
	return r.handlers[subj]
}

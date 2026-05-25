package queue

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/queue/handlers"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const (
	msgsPerLoop    = 10
	pullMaxWaiting = 128
)

type NATS struct {
	nc   *nats.Conn
	js   nats.JetStreamContext
	conf config.Nats
}

func New(conf config.Config) *NATS {
	nc, err := nats.Connect(conf.Nats.URL, nats.UserInfo(conf.Nats.Username, conf.Nats.Password))
	if err != nil {
		zap.L().Fatal("Failed to connect to NATS", zap.Error(err))
	}

	zap.L().Info("Connected to NATS", zap.String("url", conf.Nats.URL))

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		zap.L().Fatal("Failed to connect to NATS", zap.Error(err))
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     conf.Nats.StreamName,
		Subjects: conf.Nats.StreamSubjects,
		Storage:  nats.FileStorage,
	})
	if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		zap.L().Fatal("Failed to add NATS stream", zap.Error(err))
	}

	return &NATS{
		nc:   nc,
		js:   js,
		conf: conf.Nats,
	}
}

func (q *NATS) Publish(subject string, payload any, uid uuid.UUID) error {
	if q.js == nil {
		return nats.ErrConnectionClosed
	}

	data, err := json.Marshal(payload)
	if err != nil {
		zap.L().Error("Failed to marshal payload for publishing", zap.Error(err))
		return err
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header: nats.Header{
			"idemp": []string{uuid.New().String()},
			"uid":   []string{uid.String()},
		},
	}

	_, err = q.js.PublishMsg(msg)
	if err != nil {
		zap.L().Error(
			"Failed to publish message",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return err
	}

	zap.L().Debug(
		"Published message",
		zap.String("subject", subject),
	)
	return nil
}

func (q *NATS) ProcessMsgs(ctx context.Context, router handlers.Router) {
	sub, err := q.js.PullSubscribe(
		q.conf.PullSubject,
		q.conf.PullDurable,
		nats.PullMaxWaiting(pullMaxWaiting),
		nats.AckExplicit(),
	)
	if err != nil {
		zap.L().Error("Failed to pull messages", zap.Error(err))
		return
	}

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Shutting down consumer")

			if err = sub.Unsubscribe(); err != nil {
				zap.L().Error("Failed to unsubscribe", zap.Error(err))
				return
			}

			if err = q.nc.Drain(); err != nil {
				zap.L().Error("Failed to drain queue", zap.Error(err))
				return
			}

			return
		default:
			msgs, err := sub.Fetch(msgsPerLoop)
			if err != nil && !errors.Is(err, nats.ErrTimeout) {
				zap.L().Error("Failed to fetch messages", zap.Error(err))
				continue
			}

			for _, msg := range msgs {
				q.process(&router, msg)
			}
		}
	}
}

func (q *NATS) Close() error {
	if q.nc != nil {
		return q.nc.Drain()
	}
	return nil
}

func (q *NATS) process(router *handlers.Router, msg *nats.Msg) {
	hdl := router.Get(msg.Subject)
	if err := hdl.Handle(msg); err != nil {
		zap.L().Error("Failed to handle event", zap.Error(err))

		if err = msg.Nak(); err != nil {
			zap.L().Error("Failed to nak message", zap.Error(err))
		}
		return
	}

	if err := msg.Ack(); err != nil {
		zap.L().Error("Failed to ack message", zap.Error(err))
	}
}

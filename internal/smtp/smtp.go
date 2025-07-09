package smtp

import (
	"github.com/JMURv/golang-clean-template/internal/config"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type EmailServer struct {
	server       string
	port         int
	user         string
	pass         string
	admin        string
	serverConfig config.ServerConfig
}

func New(conf config.Config) *EmailServer {
	return &EmailServer{
		server:       conf.Email.Server,
		port:         conf.Email.Port,
		user:         conf.Email.User,
		pass:         conf.Email.Pass,
		admin:        conf.Email.Admin,
		serverConfig: conf.Server,
	}
}

func (s *EmailServer) GetMessageBase(subject, toEmail string) *gomail.Message {
	m := gomail.NewMessage()
	m.SetHeader("From", s.user)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", subject)
	return m
}

func (s *EmailServer) Send(m *gomail.Message) error {
	d := gomail.NewDialer(s.server, s.port, s.user, s.pass)
	if err := d.DialAndSend(m); err != nil {
		zap.L().Error(
			"Failed to send an email",
			zap.Error(err),
		)
		return err
	}
	return nil
}

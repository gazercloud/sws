package mailserver

import (
	"github.com/emersion/go-smtp"
	"github.com/gazercloud/sws/logger"
	"time"
)

type MailServer struct {
	s *smtp.Server
}

func NewMailServer() *MailServer {
	var c MailServer
	return &c
}

func (c *MailServer) Start() {
	be := &Backend{}
	c.s = smtp.NewServer(be)
	go c.serving()
}

func (c *MailServer) Stop() {
}

func (c *MailServer) serving() {
	c.s.Addr = ":25"
	c.s.Domain = "ipoluyanov.com"
	c.s.ReadTimeout = 10 * time.Second
	c.s.WriteTimeout = 10 * time.Second
	c.s.MaxMessageBytes = 1024 * 1024
	c.s.MaxRecipients = 50
	c.s.AllowInsecureAuth = true

	logger.Println("Starting server at", c.s.Addr)
	if err := c.s.ListenAndServe(); err != nil {
		logger.Println("Mail Server error: ", err)
	}
}

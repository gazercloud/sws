package mailserver

import (
	"errors"
	"github.com/emersion/go-smtp"
	"github.com/gazercloud/sws/logger"
)

// The Backend implements SMTP server methods.
type Backend struct{}

// Login handles a login command with username and password.
func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	logger.Println("MailServer Login", username, password)
	if username != "username" || password != "password" {
		return nil, errors.New("invalid username or password")
	}
	return &Session{}, nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	logger.Println("MailServer Login anonymous")
	return &Session{}, nil
}

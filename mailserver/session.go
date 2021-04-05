package mailserver

import (
	"github.com/emersion/go-smtp"
	"github.com/gazercloud/sws/logger"
	"io"
	"io/ioutil"
)

// A Session is returned after successful login.
type Session struct{}

func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	logger.Println("Mail from:", from)
	return nil
}

func (s *Session) Rcpt(to string) error {
	logger.Println("Rcpt to:", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		logger.Println("Data:", string(b))
	}
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	logger.Println("Logout")
	return nil
}

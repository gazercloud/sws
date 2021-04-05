package app

import (
	"fmt"
	"github.com/gazercloud/sws/httpserver"
	"github.com/gazercloud/sws/logger"
	"github.com/gazercloud/sws/mailserver"
)

var httpServer *httpserver.HttpServer
var mailServer *mailserver.MailServer

func Start() {
	TuneFDs()
	httpServer = httpserver.NewHttpServer()
	httpServer.Start()

	mailServer = mailserver.NewMailServer()
	mailServer.Start()
}

func Stop() {
	httpServer.Stop()
	mailServer.Stop()
}

func RunDesktop() {
	logger.Println("Running as console application")
	Start()
	fmt.Scanln()
	logger.Println("Console application exit")
}

func RunAsService() error {
	Start()
	return nil
}

func StopService() {
	Stop()
}

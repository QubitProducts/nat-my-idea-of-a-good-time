package main

import (
	"flag"
	"fmt"
	"net/smtp"

	"github.com/golang/glog"
)

var (
	smtpServer   string
	smtpUsername string
	smtpPassword string
	smtpSource   string
	smtpTarget   string
)

func init() {
	flag.StringVar(&smtpServer, "smtp-server", getEnv("NAT_SMTP_SERVER", ""), "SMTP server use for alert sending")
	flag.StringVar(&smtpUsername, "smtp-username", getEnv("NAT_SMTP_USERNAME", ""), "SMTP server username")
	flag.StringVar(&smtpPassword, "smtp-password", getEnv("NAT_SMTP_PASSWORD", ""), "SMTP server password")
	flag.StringVar(&smtpSource, "smtp-source", getEnv("NAT_SMTP_SOURCE", ""), "Email address to send alerts as")
	flag.StringVar(&smtpTarget, "smtp-target", getEnv("NAT_SMTP_TARGET", ""), "Email address to send alerts to")
}

func makeEmailAction() Action {
	if smtpServer+smtpUsername+smtpPassword+smtpSource+smtpTarget == "" {
		glog.Infof("Skipping email action due to absent configuration")
		return nil
	}

	if smtpServer == "" ||
		smtpUsername == "" ||
		smtpPassword == "" ||
		smtpSource == "" ||
		smtpTarget == "" {
		glog.Fatalln("Email configuration incomplete")
	}

	return makeAction(sendEmail)
}

func sendEmail(checkError error) error {
	glog.Infoln("Sending alert email")

	msg := []byte(fmt.Sprintf(`From: %v
To: %v
Subject: NAT FAILURE

HEY YOUR NAT'S BROKEN! I FAILED IT OVER FOR YOU (HOPEFULLY)

My health check failed with the error %v

Yours, always,

The NAT King
`, smtpTarget, smtpSource, checkError))

	var err error
	if dryRun {
		glog.Infof("Would be sending email: %v", msg)
	} else {
		err = smtp.SendMail(smtpServer, smtp.CRAMMD5Auth(smtpUsername, smtpPassword), smtpSource, []string{smtpTarget}, msg)
	}

	return err
}

package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/gomail.v2"
	"log"
)

type MailService struct {
	*gomail.Dialer
	open bool
}

func newMailService(conf config) (*MailService, error) {
	d := gomail.NewDialer(conf.mailHost, conf.mailPort, conf.username, conf.password)
	return &MailService{d, false}, nil
}

func (ms *MailService) serve(ctx context.Context, ch <-chan *gomail.Message) error {
	sender, err := ms.Dial()
	if err != nil {
		return errors.Wrap(err, "dials to smtp server err")
	}
	ms.open = true
	log.Printf("connected to smtp server: %s\n", ms.Host)

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("can`t get value from chan")
			}
			if !ms.open {
				if sender, err = ms.Dial(); err != nil {
					return errors.Wrap(err, "dials to smtp server err")
				}
				ms.open = true
			}
			if err := gomail.Send(sender, msg); err != nil {
				log.Printf("send email err: %v", err)
			}
			log.Printf("email to %s is send success", msg.GetHeader("To"))
		case <-ctx.Done():
			if ms.open {
				if err := sender.Close(); err != nil {
					return errors.Wrap(err, "closing sender err")
				}
				ms.open = false
			}
			return nil
		}
	}
}

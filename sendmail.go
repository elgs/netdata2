package main

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
)

func SendMail(subject, body string, to ...string) error {
	return sendMail(service.MailHost, strconv.Itoa(service.MailPort), service.MailUsername, service.MailPassword, subject, body, "uprun@uprun.io", to...)
}

func sendMail(host, port, username, password, subject, body, from string, to ...string) error {
	// Connect to the remote SMTP server.
	c, err := smtp.Dial(host + ":" + port)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{InsecureSkipVerify: true}
		if err = c.StartTLS(config); err != nil {
			fmt.Println(err)
			return err
		}
	}

	if ok, _ := c.Extension("AUTH"); ok {
		a := smtp.PlainAuth("", username, password, host)
		if err = c.Auth(a); err != nil {
			fmt.Println(err)
			return err
		}
	}

	var message string
	message += "Subject:" + subject + "\r\n"
	// Set the sender and recipient first
	if err := c.Mail(from); err != nil {
		fmt.Println(err)
		return err
	}
	message += "From:" + from + "\r\n"

	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			fmt.Println(err)
		}
		message += "To:" + rcpt + "\r\n"
	}

	// Send the email body.
	message += "\r\n\r\n" + body
	wc, err := c.Data()
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = fmt.Fprintf(wc, message)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = wc.Close()
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

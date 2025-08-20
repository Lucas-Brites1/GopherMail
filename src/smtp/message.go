package smtp

import (
	"fmt"
	"strings"
	"time"
)

func (c *SMTPClient) from() error {
	if err := c.writeFromConn("MAIL FROM:<%s>", c.Helper.Email.Header.From); err != nil {
		return err
	}

	_, _, err := c.readFromConn(OK)
	return err
}

func (c *SMTPClient) to() error {
	allRecipients := make([]string, 0)
	allRecipients = append(allRecipients, c.Helper.Email.Header.To...)
	allRecipients = append(allRecipients, c.Helper.Email.Header.Cc...)
	allRecipients = append(allRecipients, c.Helper.Email.Header.Bcc...)

	for _, recipient := range allRecipients {
		if err := c.writeFromConn("RCPT TO:<%s>", recipient); err != nil {
			return err
		}

		if _, _, err := c.readFromConn(OK); err != nil {
			return err
		}
	}

	return nil
}

func (c *SMTPClient) data() error {
	if err := c.writeFromConn("DATA"); err != nil {
		return err
	}

	if _, _, err := c.readFromConn(StartMailInput); err != nil {
		return err
	}

	c.Buffer.Write(fmt.Sprintf("Subject: %s\r\n", c.Helper.Email.Content.Subject))
	c.Buffer.Write(fmt.Sprintf("From: %s\r\n", c.Helper.Email.Header.From))

	if len(c.Helper.Email.Header.To) > 0 {
		c.Buffer.Write(fmt.Sprintf("To: %s\r\n", strings.Join(c.Helper.Email.Header.To, ", ")))
	}

	if len(c.Helper.Email.Header.Cc) > 0 {
		c.Buffer.Write(fmt.Sprintf("Cc: %s\r\n", strings.Join(c.Helper.Email.Header.Cc, ", ")))
	}

	c.Buffer.Write(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	c.Buffer.Write("\r\n")

	c.Buffer.Write(c.Helper.Email.Content.Body)
	c.Buffer.Write("\r\n.\r\n")

	finalMsg := c.Buffer.Get()

	if err := c.writeFromConn(finalMsg); err != nil {
		return err
	}

	c.Buffer.Reset()
	return nil
}

func (c *SMTPClient) SendMail() error {
	if err := c.from(); err != nil {
		return err
	}

	if err := c.to(); err != nil {
		return err
	}

	if err := c.data(); err != nil {
		return err
	}

	if err := c.quit(); err != nil {
		return err
	}

	return nil
}

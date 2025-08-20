package smtp

import (
	"fmt"
	"strings"
)

type SMTPCode int

const (
	BannerReady         SMTPCode = 220
	ServiceClosing      SMTPCode = 221
	HelpMessage         SMTPCode = 214
	ServiceNotAvailable SMTPCode = 421

	OK           SMTPCode = 250
	UserNotLocal SMTPCode = 251
	CannotVerify SMTPCode = 252

	StartMailInput SMTPCode = 354

	AuthLoginSuccess SMTPCode = 235
	AuthRequired     SMTPCode = 334

	MailboxBusy         SMTPCode = 450
	LocalError          SMTPCode = 451
	InsufficientStorage SMTPCode = 452

	SyntaxError            SMTPCode = 500
	CmdNotRecognized       SMTPCode = 500
	CmdParamNotRecognized  SMTPCode = 501
	MustIssueStartTLS      SMTPCode = 530
	MailboxUnavailable     SMTPCode = 550
	UserNotLocalTryForward SMTPCode = 551
	ExceededStorage        SMTPCode = 552
	MailboxNameNotAllowed  SMTPCode = 553
	TransactionFailed      SMTPCode = 554
)

func (c *SMTPClient) getAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

/*
func (c *SMTPClient) checkCode(code SMTPCode, expected SMTPCode) error {
	if code != expected {
		return fmt.Errorf("SMTP: unexpected code. Expected %d, got %d", expected, code)
	}
	return nil
}
*/

func (c *SMTPClient) readFromConn(expected SMTPCode) (SMTPCode, string, error) {
	gotCode, msg, err := c.Text.ReadResponse(int(expected))
	if err != nil {
		return 0, "", err
	}
	return SMTPCode(gotCode), msg, nil
}

func (c *SMTPClient) writeConnWrapper(message ...string) string {
	return strings.Join(message, "")
}

func (c *SMTPClient) writeFromConn(message ...string) error {
	str := c.writeConnWrapper(message...)
	if err := c.Text.PrintfLine(str); err != nil {
		return err
	}
	return nil
}

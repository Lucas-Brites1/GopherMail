package smtp

import (
	"crypto/tls"
	"net/textproto"
	"time"

	"github.com/Lucas-Brites1/GopherMail/src/models"
	"github.com/Lucas-Brites1/GopherMail/src/utils"
)

type SMTPClient struct {
	Host      string
	Port      int
	Conn      *tls.Conn
	TLSConfig *tls.Config
	Timeout   time.Duration

	Text   *textproto.Conn
	Buffer *utils.Buffer_t

	Helper struct {
		Config models.Config
		Email  models.Email
	}
}

func NewSMTPClient(host string, port int, useTLS bool) *SMTPClient {
	return &SMTPClient{
		Host:    host,
		Port:    port,
		Timeout: 10 * time.Second,
		Buffer:  utils.NewBuffer(50, 3),
		TLSConfig: &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         host,
		},
	}
}

package smtp

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/textproto"
)

func (c *SMTPClient) EstablishTLS() error {
	addr := c.getAddr()
	if addr == "" {
		return fmt.Errorf("something went wrong trying to get the address connection")
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: c.Timeout}, "tcp", addr, c.TLSConfig)
	if err != nil {
		return err
	}

	err = conn.Handshake()
	if err != nil {
		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	c.Conn = conn
	c.Text = textproto.NewConn(c.Conn)

	return nil
}

func (c *SMTPClient) SMTPHandshake() error {
	if err := c.initialBanner(); err != nil {
		return err
	}

	if err := c.ehlo(); err != nil {
		return err
	}

	if err := c.auth(); err != nil {
		return err
	}

	return nil
}

func (c *SMTPClient) Connect() error {
	if err := c.EstablishTLS(); err != nil {
		return err
	}

	if err := c.SMTPHandshake(); err != nil {
		return err
	}

	return nil
}

func (c *SMTPClient) initialBanner() error {
	_, _, err := c.readFromConn(BannerReady)
	if err != nil {
		return fmt.Errorf("failed to read banner: %w", err)
	}

	return nil
}

func (c *SMTPClient) ehlo() error {
	if err := c.writeFromConn(fmt.Sprintf("EHLO %s", c.Host)); err != nil {
		return err
	}

	_, _, err := c.readFromConn(OK)
	return err
}

func (c *SMTPClient) auth() error {
	if err := c.writeFromConn("AUTH LOGIN"); err != nil {
		return err
	}

	if _, _, err := c.readFromConn(AuthRequired); err != nil {
		return err
	}

	b64_username := base64.StdEncoding.EncodeToString([]byte(c.Helper.Config.SMTPUser))
	if err := c.writeFromConn(b64_username); err != nil {
		return err
	}

	if _, _, err := c.readFromConn(AuthRequired); err != nil {
		return err
	}

	b64_password := base64.StdEncoding.EncodeToString([]byte(c.Helper.Config.SMTPPassword))
	if err := c.writeFromConn(b64_password); err != nil {
		return err
	}

	_, _, err := c.readFromConn(AuthLoginSuccess)
	return err
}

func (c *SMTPClient) quit() error {
	if err := c.writeFromConn("QUIT"); err != nil {
		return err
	}

	_, _, err := c.readFromConn(ServiceClosing)
	return err
}

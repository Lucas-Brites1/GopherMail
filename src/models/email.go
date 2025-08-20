package models

type Header_t struct {
	From string
	To   []string
	Cc   []string
	Bcc  []string
}

type Content_t struct {
	Subject string
	Body    string
}

type Email struct {
	Header  Header_t
	Content Content_t
}

func CreateEmail(recipient []string, subject, body string) (*Email, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return &Email{
		Content: Content_t{
			Subject: subject,
			Body:    body,
		},
		Header: Header_t{
			To:   recipient,
			From: cfg.SMTPUser,
			Cc:   nil,
			Bcc:  nil,
		},
	}, nil
}

func (em *Email) ApplyCc(emailsCc ...string) {
	em.Header.Cc = append(em.Header.Cc, emailsCc...)
}

func (em *Email) ApplyBcc(emailsBcc ...string) {
	em.Header.Bcc = append(em.Header.Bcc, emailsBcc...)
}

package gophermail

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
)

type MailInfos struct {
	From string
	To   []string
	Cc   []string
	Bcc  []string
}

type MailContent struct {
	Subject string
	Body    string
}

type Config struct {
	Infos   MailInfos
	Content MailContent

	SmtpServer string
	SmtpPort   int
	PasswordTK string
	Retry      bool
}

func (c *Config) EstablishTLSConn() (conn *tls.Conn, err error) {
	address := fmt.Sprintf("%s:%d", c.SmtpServer, c.SmtpPort)
	tlsconfig := &tls.Config{
		ServerName: c.SmtpServer,
	}

	conn, err = tls.Dial("tcp", address, tlsconfig)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *Config) GetSMTPAuth() smtp.Auth {
	return smtp.PlainAuth("", c.Infos.From, c.PasswordTK, c.SmtpServer)
}

func (c *Config) CreateSMTPClient() (*smtp.Client, error) {
	tlsConn, err := c.EstablishTLSConn()
	if err != nil {
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	client, err := smtp.NewClient(tlsConn, c.SmtpServer)
	if err != nil {
		return nil, err
	}

	if err := client.Auth(c.GetSMTPAuth()); err != nil {
		return nil, err
	}

	return client, nil
}

func NewConfig(shouldRetry bool) *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	smtpServer := os.Getenv("SMTP_SERVER")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		log.Fatal("Invalid SMTP_PORT: ", err)
	}

	mailPass := os.Getenv("MAIL_PASS")
	mailFrom := os.Getenv("MAIL_FROM")

	return &Config{
		SmtpServer: smtpServer,
		SmtpPort:   smtpPort,
		PasswordTK: mailPass,
		Retry:      shouldRetry,
		Infos: MailInfos{
			From: mailFrom,
			To:   nil,
		},
		Content: MailContent{
			Subject: "",
			Body:    "",
		},
	}
}

type IndividualEmail struct {
	ID      int
	From    string
	To      string
	Cc      []string
	Bcc     []string
	Subject string
	Body    string
}

type EmailResult struct {
	Email    *IndividualEmail
	Success  bool
	Error    error
	Time     time.Duration
	SentAt   time.Time
	Attempts int
}

type EmailWorkerPool struct {
	Config      *Config
	NumWorkers  int
	JobQueue    chan *IndividualEmail
	ResultQueue chan EmailResult
	wg          sync.WaitGroup
	quit        chan struct{}
	mu          sync.Mutex
	counter     int64
	started     bool
	RetrySystem *EmailRetrySystem
}

type EmailRetrySystem struct {
	maxRetries     int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	multiplier     float64
}

func NewRetrySystem(maxRetries int) *EmailRetrySystem {
	return &EmailRetrySystem{
		maxRetries:     maxRetries,
		initialBackoff: time.Second,
		maxBackoff:     60 * time.Second,
		multiplier:     2.0,
	}
}

func (ers *EmailRetrySystem) shouldRetry(err error) bool {
	errStr := err.Error()

	retryableErrors := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network unreachable",
		"TLS handshake failed",
		"connection reset",
		"server busy",
		"dial tcp",
		"no such host",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryable) {
			return true
		}
	}

	nonRetryableErrors := []string{
		"invalid sender format",
		"invalid recipient format",
		"validation failed",
		"authentication failed",
		"invalid credentials",
		"535",
		"550",
	}

	for _, nonRetryable := range nonRetryableErrors {
		if strings.Contains(strings.ToLower(errStr), nonRetryable) {
			return false
		}
	}

	return true
}

func (ers *EmailRetrySystem) SendWithRetry(ewp *EmailWorkerPool, email *IndividualEmail) (error, int) {
	var lastErr error
	backoff := ers.initialBackoff

	for attempt := 1; attempt <= ers.maxRetries; attempt++ {
		fmt.Printf("Attempt %d/%d for email %d (%s)\n",
			attempt, ers.maxRetries, email.ID, email.To)

		err := ewp.sendEmail(email)
		if err == nil {
			if attempt > 1 {
				fmt.Printf("Email %d succeeded on attempt %d\n", email.ID, attempt)
			}
			return nil, attempt
		}

		lastErr = err
		fmt.Printf("Attempt %d failed for email %d: %v\n", attempt, email.ID, err)

		if attempt < ers.maxRetries {
			if ers.shouldRetry(err) {
				time.Sleep(backoff)

				backoff = time.Duration(float64(backoff) * ers.multiplier)
				if backoff > ers.maxBackoff {
					backoff = ers.maxBackoff
				}
			} else {
				fmt.Printf("Error is not retryable for email %d: %v\n", email.ID, err)
				break
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", ers.maxRetries, lastErr), ers.maxRetries
}

func NewEmailWorkerPool(config *Config, numWorkers int, maxRetries int) *EmailWorkerPool {
	retrySystem := NewRetrySystem(maxRetries)

	return &EmailWorkerPool{
		Config:      config,
		NumWorkers:  numWorkers,
		JobQueue:    make(chan *IndividualEmail, numWorkers*2),
		ResultQueue: make(chan EmailResult, numWorkers*2),
		quit:        make(chan struct{}),
		started:     false,
		RetrySystem: retrySystem,
	}
}

func (ewp *EmailWorkerPool) worker(id int, jobs <-chan *IndividualEmail, results chan<- EmailResult) {
	for job := range jobs {
		start := time.Now()
		fmt.Printf("Worker %d processing email %d to %s\n", id, job.ID, job.To)

		var err error
		var attempts int

		if ewp.Config.Retry && ewp.RetrySystem != nil {
			err, attempts = ewp.RetrySystem.SendWithRetry(ewp, job)
		} else {
			err = ewp.sendEmail(job)
			attempts = 1
		}

		duration := time.Since(start)

		results <- EmailResult{
			Email:    job,
			Success:  err == nil,
			Error:    err,
			Time:     duration,
			SentAt:   time.Now(),
			Attempts: attempts,
		}

		if err == nil {
			if attempts > 1 {
				fmt.Printf("Worker %d: Email %d sent to %s in %v (after %d attempts)\n",
					id, job.ID, job.To, duration, attempts)
			} else {
				fmt.Printf("Worker %d: Email %d sent to %s in %v\n",
					id, job.ID, job.To, duration)
			}
		} else {
			fmt.Printf("Worker %d: Email %d to %s failed after %d attempts: %v\n",
				id, job.ID, job.To, attempts, err)
		}
	}

	fmt.Printf("Worker %d finished\n", id)
}

func (ewp *EmailWorkerPool) Start() {
	ewp.mu.Lock()
	defer ewp.mu.Unlock()

	if ewp.started {
		fmt.Println("Worker pool already started")
		return
	}

	retryMsg := "without retry"
	if ewp.Config.Retry && ewp.RetrySystem != nil {
		retryMsg = fmt.Sprintf("with retry (max %d attempts)", ewp.RetrySystem.maxRetries)
	}

	fmt.Printf("Starting worker pool with %d workers %s...\n", ewp.NumWorkers, retryMsg)

	for w := 1; w <= ewp.NumWorkers; w++ {
		ewp.wg.Add(1)
		go func(workerID int) {
			defer ewp.wg.Done()
			ewp.worker(workerID, ewp.JobQueue, ewp.ResultQueue)
		}(w)
	}

	ewp.started = true
	fmt.Println("Worker pool started successfully")
}

func (ewp *EmailWorkerPool) Stop() {
	ewp.mu.Lock()
	defer ewp.mu.Unlock()

	if !ewp.started {
		fmt.Println("Worker pool not started")
		return
	}

	fmt.Println("Stopping worker pool...")
	close(ewp.JobQueue)
	ewp.wg.Wait()
	close(ewp.ResultQueue)
	ewp.started = false
	fmt.Println("Worker pool stopped successfully")
}

func (ewp *EmailWorkerPool) getNextID() int {
	return int(atomic.AddInt64(&ewp.counter, 1))
}

func (ewp *EmailWorkerPool) CreateEmail(emailTo, subject, body string, cc, bcc []string) *IndividualEmail {
	return &IndividualEmail{
		ID:      ewp.getNextID(),
		From:    ewp.Config.Infos.From,
		To:      emailTo,
		Cc:      cc,
		Bcc:     bcc,
		Subject: subject,
		Body:    body,
	}
}

func (ewp *EmailWorkerPool) CreateSimpleEmail(emailTo, subject, body string) *IndividualEmail {
	return ewp.CreateEmail(emailTo, subject, body, nil, nil)
}

func (ewp *EmailWorkerPool) CreateEmailWithCC(emailTo, subject, body string, cc []string) *IndividualEmail {
	return ewp.CreateEmail(emailTo, subject, body, cc, nil)
}

func (ewp *EmailWorkerPool) CreateEmailWithBCC(emailTo, subject, body string, bcc []string) *IndividualEmail {
	return ewp.CreateEmail(emailTo, subject, body, nil, bcc)
}

func (ewp *EmailWorkerPool) AddEmail(email *IndividualEmail) error {
	ewp.mu.Lock()
	defer ewp.mu.Unlock()

	if !ewp.started {
		return fmt.Errorf("worker pool not started")
	}

	select {
	case ewp.JobQueue <- email:
		fmt.Printf("Email %d added to queue (%s)\n", email.ID, email.To)
		return nil
	case <-ewp.quit:
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("job queue is full")
	}
}

func (ewp *EmailWorkerPool) ProcessEmails(emails []*IndividualEmail) []EmailResult {
	ewp.Start()

	go func() {
		for _, email := range emails {
			if err := ewp.AddEmail(email); err != nil {
				fmt.Printf("Error adding email %d: %v\n", email.ID, err)
			}
		}
		time.Sleep(100 * time.Millisecond)
		ewp.Stop()
	}()

	var results []EmailResult
	for result := range ewp.ResultQueue {
		results = append(results, result)
	}

	return results
}

func (ewp *EmailWorkerPool) SetRetryConfig(maxRetries int, initialBackoff, maxBackoff time.Duration) {
	ewp.mu.Lock()
	defer ewp.mu.Unlock()

	ewp.RetrySystem = &EmailRetrySystem{
		maxRetries:     maxRetries,
		initialBackoff: initialBackoff,
		maxBackoff:     maxBackoff,
		multiplier:     2.0,
	}
	ewp.Config.Retry = maxRetries > 0
}

func defineRecipients(client *smtp.Client, email *IndividualEmail) error {
	allRecipients := []string{email.To}

	for _, cc := range email.Cc {
		if cc != "" && strings.Contains(cc, "@") {
			allRecipients = append(allRecipients, cc)
		}
	}

	for _, bcc := range email.Bcc {
		if bcc != "" && strings.Contains(bcc, "@") {
			allRecipients = append(allRecipients, bcc)
		}
	}

	for _, recipient := range allRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("error setting recipient %s: %w", recipient, err)
		}
	}

	return nil
}

func validateEmail(email *IndividualEmail) error {
	if email == nil {
		return fmt.Errorf("email cannot be nil")
	}
	if email.From == "" {
		return fmt.Errorf("sender (From) is required")
	}
	if email.To == "" {
		return fmt.Errorf("recipient (To) is required")
	}
	if email.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if email.Body == "" {
		return fmt.Errorf("body is required")
	}

	if !strings.Contains(email.From, "@") {
		return fmt.Errorf("invalid sender format: %s", email.From)
	}
	if !strings.Contains(email.To, "@") {
		return fmt.Errorf("invalid recipient format: %s", email.To)
	}

	return nil
}

func buildMessage(email *IndividualEmail) []byte {
	var message strings.Builder
	message.WriteString("From: " + email.From + "\r\n")
	message.WriteString("To: " + email.To + "\r\n")

	if len(email.Cc) > 0 {
		var validCcs []string
		for _, cc := range email.Cc {
			if cc != "" {
				validCcs = append(validCcs, cc)
			}
		}
		if len(validCcs) > 0 {
			message.WriteString("Cc: " + strings.Join(validCcs, ", ") + "\r\n")
		}
	}

	message.WriteString("Subject: " + email.Subject + "\r\n")
	message.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	message.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")

	message.WriteString("\r\n")

	message.WriteString(email.Body)

	return []byte(message.String())
}

func (ewp *EmailWorkerPool) sendEmail(email *IndividualEmail) error {

	if err := validateEmail(email); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	client, err := ewp.Config.CreateSMTPClient()
	if err != nil {
		return fmt.Errorf("error creating SMTP client: %w", err)
	}
	defer client.Quit()

	if err := client.Mail(email.From); err != nil {
		return fmt.Errorf("error setting sender: %w", err)
	}

	err = defineRecipients(client, email)
	if err != nil {
		return fmt.Errorf("error setting recipients: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("error starting data transmission: %w", err)
	}

	message := buildMessage(email)
	if _, err = w.Write(message); err != nil {
		w.Close()
		return fmt.Errorf("error writing message: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("error finalizing email sending: %w", err)
	}

	return nil
}

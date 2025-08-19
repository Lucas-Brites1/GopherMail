# 🚀 GopherMail

High-performance concurrent SMTP email service with intelligent retry system for Go applications.

## ⚡ Features

- **🚀 3-5x faster** than sequential email sending
- **🔄 Intelligent retry** with exponential backoff
- **👷 Worker pool** architecture for controlled concurrency
- **🛡️ Thread-safe** operations with comprehensive error handling
- **📊 Detailed metrics** and logging for production monitoring
- **⚙️ Configurable** retry policies and worker counts
- **📧 Full email support** including CC, BCC, and custom headers

## 📈 Performance

```
Sequential Processing:  100 emails = ~200 seconds (3+ minutes)
GopherMail Worker Pool: 100 emails = ~30 seconds  (3-5x faster!)
```

## 🚀 Quick Start

### Installation

```bash
go get github.com/Lucas-Brites1/GopherMail
```

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/lucas-Brites1/GopherMail"
)

func main() {
    // Create configuration with retry enabled
    config := gophermail.NewConfig(true)
    
    // Create worker pool (3 workers, max 3 retry attempts)
    pool := gophermail.NewEmailWorkerPool(config, 3, 3)
    
    // Create emails
    emails := []*gophermail.IndividualEmail{
        pool.CreateSimpleEmail("user@example.com", "Hello", "Welcome to GopherMail!"),
        pool.CreateEmailWithCC("manager@example.com", "Report", "Monthly report", []string{"team@example.com"}),
    }
    
    // Send emails concurrently
    results := pool.ProcessEmails(emails)
    
    // Check results
    for _, result := range results {
        if result.Success {
            fmt.Printf("✅ Email sent to %s in %v\n", result.Email.To, result.Time)
        } else {
            fmt.Printf("❌ Failed to send to %s: %v\n", result.Email.To, result.Error)
        }
    }
}
```

## 📋 Setup

1. **Copy environment template:**
   ```bash
   cp .env.example .env
   ```

2. **Configure your SMTP settings in `.env`:**
   ```env
   SMTP_SERVER=smtp.gmail.com
   SMTP_PORT=587
   MAIL_FROM=your-email@gmail.com
   MAIL_PASS=your-app-password
   ```

3. **Run example:**
   ```bash
   go run examples/basic_usage.go
   ```

## 💻 Examples

### Simple Email Sending

```go
config := gophermail.NewConfig(true)
pool := gophermail.NewEmailWorkerPool(config, 3, 3)

// Single email
email := pool.CreateSimpleEmail(
    "user@example.com",
    "Welcome!",
    "Thank you for joining our service!",
)

results := pool.ProcessEmails([]*gophermail.IndividualEmail{email})
```

### Bulk Email Campaign

```go
// Newsletter to 1000 subscribers
subscribers := []string{
    "user1@example.com", "user2@example.com", // ... 1000 emails
}

var emails []*gophermail.IndividualEmail
for _, subscriber := range subscribers {
    email := pool.CreateSimpleEmail(
        subscriber,
        "Weekly Newsletter",
        "Check out this week's updates...",
    )
    emails = append(emails, email)
}

// Process all emails concurrently
results := pool.ProcessEmails(emails)
fmt.Printf("Sent %d/%d emails successfully\n", 
    countSuccessful(results), len(results))
```

### Advanced Configuration

```go
// Custom retry configuration
pool.SetRetryConfig(
    5,                    // Max 5 retry attempts
    2*time.Second,        // Initial backoff: 2 seconds
    60*time.Second,       // Max backoff: 60 seconds
)

// Email with CC and BCC
email := pool.CreateEmail(
    "primary@example.com",           // To
    "Important Update",              // Subject
    "This is an important message",  // Body
    []string{"cc1@example.com", "cc2@example.com"}, // CC
    []string{"bcc@example.com"},     // BCC
)
```

### Error Handling and Monitoring

```go
results := pool.ProcessEmails(emails)

// Detailed analysis
var successful, failed, retried int
for _, result := range results {
    if result.Success {
        successful++
        if result.Attempts > 1 {
            retried++
            fmt.Printf("📧 %s succeeded after %d attempts\n", 
                result.Email.To, result.Attempts)
        }
    } else {
        failed++
        fmt.Printf("❌ %s failed after %d attempts: %v\n", 
            result.Email.To, result.Attempts, result.Error)
    }
}

fmt.Printf(`
📊 Campaign Results:
✅ Successful: %d
❌ Failed: %d  
🔄 Required retry: %d
`, successful, failed, retried)
```

## ⚙️ Configuration

### Worker Pool Settings

| Parameter | Recommended | Description |
|-----------|-------------|-------------|
| **Workers** | 3-5 | Number of concurrent workers |
| **Max Retries** | 3 | Maximum retry attempts |
| **Buffer Size** | workers*2 | Queue buffer size |

### SMTP Provider Settings

| Provider | Workers | Notes |
|----------|---------|-------|
| **Gmail** | 2-3 | Rate limited |
| **Outlook** | 2-3 | Conservative limits |
| **SendGrid** | 10+ | High throughput |
| **Custom SMTP** | 5-10 | Test and adjust |

## 🔄 Retry System

The intelligent retry system automatically handles:

- **🌐 Network failures** (connection refused, timeout)
- **🔧 Temporary SMTP errors** (server busy, rate limiting)
- **🚫 Permanent failures** (invalid email, auth errors) - no retry

### Backoff Strategy

```
Attempt 1: Immediate
Attempt 2: Wait 1s
Attempt 3: Wait 2s  
Attempt 4: Wait 4s
Attempt 5: Wait 8s (exponential backoff)
```

## 📊 Use Cases

### Perfect For:
- 📧 **Newsletter campaigns** (1000+ emails)
- 🔔 **Notification systems** (real-time alerts)
- 📈 **Transactional emails** (receipts, confirmations)
- 🎯 **Marketing campaigns** (promotional emails)

### Performance Examples:
```
Newsletter (1000 emails):
├── Sequential: ~33 minutes
└── GopherMail: ~5-8 minutes

Notifications (100 emails):
├── Sequential: ~3.5 minutes  
└── GopherMail: ~45 seconds

Transactional (10 emails):
├── Sequential: ~20 seconds
└── GopherMail: ~4 seconds
```

## 🏗️ Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Main      │───▶│  Job Queue   │───▶│   Worker 1  │
│ Application │    │   (FIFO)     │    │   Worker 2  │
└─────────────┘    │              │    │   Worker 3  │
                   └──────────────┘    └─────────────┘
                           │                    │
                           ▼                    ▼
                   ┌──────────────┐    ┌─────────────┐
                   │ Result Queue │◀───│ Retry System│
                   │   (FIFO)     │    │ (Exponential│
                   └──────────────┘    │  Backoff)   │
                                       └─────────────┘
```

## 🧪 Testing

```bash
# Run examples
go run examples/basic_usage.go
go run examples/bulk_email.go

# Run tests (coming soon)
go test ./...
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📖 **Documentation**: Check the examples folder
- 🐛 **Issues**: Report bugs on GitHub Issues
- 💬 **Questions**: Open a GitHub Discussion

## 🔗 Related

- [Go SMTP Package](https://pkg.go.dev/net/smtp)
- [Godotenv](https://github.com/joho/godotenv)
- [Go Concurrency Patterns](https://golang.org/doc/effective_go.html#concurrency)

---

**Made with ❤️ and ☕ by [lucas-Brites1](https://github.com/Lucas-Brites1)**

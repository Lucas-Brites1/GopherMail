package main

import "time"

func main() {
	config := NewConfig(true)
	pool := NewEmailWorkerPool(config, 3, 3)
	pool.SetRetryConfig(3, time.Second, 30*time.Second)
	emails := []*IndividualEmail{
		pool.CreateSimpleEmail("lucasbrites303@gmail.com", "Teste", "Corpo do email de teste!"),
		pool.CreateSimpleEmail("lucasbrites076@gmail.com", "Teste", "Corpo do email de teste!"),
		pool.CreateSimpleEmail("cpt.victor@outlook.com", "Teste Legal", "Corpo do email de teste daora!"),
	}
	pool.ProcessEmails(emails)
}

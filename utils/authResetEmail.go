package utils

import (
	"log"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

func SendResetCodeEmail(email, code string) error {
	// Retrieve the "From" header from an environment variable
	fromEmail := os.Getenv("SMTP_USER")

	m := gomail.NewMessage()
	m.SetHeader("From", fromEmail)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Password Reset Code")

	// Set the plain text body
	m.SetBody("text/plain", "Your password reset code is: "+code)

	// Set the HTML body
	htmlBody := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Password Reset Code</title>
		<style>
			body {
				font-family: Arial, sans-serif;
				background-color: #f4f4f4;
				margin: 0;
				padding: 0;
			}
			.container {
				background-color: #ffffff;
				margin: 20px auto;
				padding: 20px;
				border-radius: 8px;
				box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
				max-width: 600px;
			}
			h1 {
				color: #333333;
			}
			p {
				color: #666666;
			}
			.code {
				font-weight: bold;
				color: #007bff;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<h1>Password Reset Code</h1>
			<p>Your password reset code is:</p>
			<p class="code">` + code + `</p>
			<p>If you did not request a password reset, please ignore this email.</p>
		</div>
	</body>
	</html>
	`
	m.AddAlternative("text/html", htmlBody)

	// Retrieve SMTP configuration from environment variables
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	// Convert the SMTP port from string to integer
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		log.Fatalf("Invalid SMTP_PORT value: %v", err)
	}

	// Create the dialer with the retrieved configuration
	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)
	return d.DialAndSend(m)
}

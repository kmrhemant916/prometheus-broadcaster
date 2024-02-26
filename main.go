package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"net/smtp"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

func main() {
	r := gin.Default()

	// NATS connection
	// nc, err := nats.Connect(nats.DefaultURL)
	nc, err := nats.Connect("nats://127.0.0.1:4222")
	
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Handler to publish message to NATS
	r.POST("/publish", func(c *gin.Context) {
		var body struct {
			Message string `json:"message"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := nc.Publish("alerts", []byte(body.Message))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish message"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Message published successfully"})
	})

	// NATS consumer
	go func() {
		// NATS subscription
		sub, err := nc.SubscribeSync("alerts")
		if err != nil {
			log.Fatal(err)
		}

		for {
			msg, err := sub.NextMsg(time.Second)
			if err != nil {
				log.Println("Error getting message:", err)
				continue
			}

			// SendGrid email alert
			err = sendEmail(string(msg.Data))
			if err != nil {
				log.Println("Error sending email:", err)
			} else {
				log.Println("Email sent successfully")
			}
		}
	}()

	r.Run(":8080")
}

func sendEmail(message string) error {
    // Set the SendGrid SMTP server and port
    server := "smtp.sendgrid.net"
    port := 587

    // Set the SendGrid username and password
    username := os.Getenv("SENDGRID_USERNAME")
    password := os.Getenv("SENDGRID_PASSWORD")

    // Set up authentication information
    auth := smtp.PlainAuth("", username, password, server)

    // Set up email content
    from := "Sender Name <sender@example.com>"
    to := []string{"recipient@example.com"}
    subject := "Alert"
    body := message

    // Compose the email message
    msg := []byte("From: " + from + "\r\n" +
        "To: " + strings.Join(to, ",") + "\r\n" +
        "Subject: " + subject + "\r\n" +
        "\r\n" +
        body + "\r\n")

    // Send the email
    err := smtp.SendMail(server+":"+strconv.Itoa(port), auth, from, to, msg)
    return err
}


package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"net/smtp"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gopkg.in/yaml.v2"
)

type Config struct {
    Service struct {
        Port string `yaml:"port"`
    } `yaml:"service"`
}

func main() {
	data, err := os.ReadFile(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatalf("error reading YAML file: %v", err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error unmarshaling YAML data: %v", err)
	}
	r := gin.Default()
	nc, err := nats.Connect(os.Getenv("NATS_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
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
	go func() {
		sub, err := nc.SubscribeSync("alerts")
		if err != nil {
			log.Fatal(err)
		}
		for {
			// Wait for a message for up to 5 seconds
			start := time.Now()
			msg, err := sub.NextMsg(5 * time.Second)
			if err != nil {
				// Check if the error is a timeout based on elapsed time
				elapsed := time.Since(start)
				if elapsed >= 5*time.Second {
					log.Println("No message received")
					continue
				}

				log.Println("Error getting message:", err)
				continue
			}
			fmt.Println([]byte(msg.Data))

			// SendGrid email alert
			// err = sendEmail(string(msg.Data))
			// if err != nil {
			// 	log.Println("Error sending email:", err)
			// } else {
			// 	log.Println("Email sent successfully")
			// }
		}
	}()

	r.Run(":"+config.Service.Port)
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


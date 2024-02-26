package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	driver "github.com/arangodb/go-driver"
	arangodbHTTP "github.com/arangodb/go-driver/http"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/nats-io/nats.go"
	"gopkg.in/yaml.v2"
)

type Config struct {
    Service struct {
        Port string `yaml:"port"`
    } `yaml:"service"`
	JWTKey string `yaml:"jwt_key"`
    ArangoDB struct {
        Host string `yaml:"host"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
    } `yaml:"arangodb"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
    endpoints, err := arangodbHTTP.NewConnection(arangodbHTTP.ConnectionConfig{
        Endpoints: []string{config.ArangoDB.Host},
    })
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Connecting to ArangoDB at %s...", config.ArangoDB.Host)
	conn, err := connectWithRetry(endpoints, config.ArangoDB.Username, config.ArangoDB.Password, 5, 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to ArangoDB")
	ctx := context.Background()
	db, err := createDatabaseIfNotExists(ctx, conn, config.ArangoDB.Database)
	if err != nil {
		log.Fatal(err)
	}
	collection, err := createCollectionIfNotExists(ctx, db, "users")
	if err != nil {
		log.Fatal(err)
	}
	authMiddleware := func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Check the token signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return config.JWTKey, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("username", claims["username"])
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
		}
	}
	r.POST("/token", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	
		// Check if the user already exists
		query := "FOR u IN users FILTER u.username == @username RETURN u"
		bindVars := map[string]interface{}{
			"username": user.Username,
		}
	
		cursor, err := collection.Database().Query(ctx, query, bindVars)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close()
	
		if cursor.HasMore() {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
	
		// Create the user document in the users collection
		_, err = collection.CreateDocument(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	
		// Generate JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": user.Username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
	
		tokenString, err := token.SignedString([]byte(config.JWTKey))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
	
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})
	nc, err := nats.Connect(os.Getenv("NATS_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	r.POST("/publish", authMiddleware, func(c *gin.Context) {
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

func connectWithRetry(endpoints driver.Connection, username, password string, retries int, delay time.Duration) (driver.Client, error) {
    var client driver.Client
    var err error
    for i := 0; i < retries; i++ {
        log.Printf("Attempting to connect (attempt %d/%d)", i+1, retries)
        client, err = driver.NewClient(driver.ClientConfig{
            Connection:     endpoints,
            Authentication: driver.BasicAuthentication(username, password),
        })
        if err == nil {
            return client, nil
        }
        log.Printf("Connection attempt failed: %v", err)
        time.Sleep(delay)
    }
    return nil, fmt.Errorf("failed to connect after %d attempts", retries)
}

func createDatabaseIfNotExists(ctx context.Context, client driver.Client, dbName string) (driver.Database, error) {
	exists, err := client.DatabaseExists(ctx, dbName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return client.CreateDatabase(ctx, dbName, nil)
	}
	return client.Database(ctx, dbName)
}

func createCollectionIfNotExists(ctx context.Context, db driver.Database, collectionName string) (driver.Collection, error) {
	exists, err := db.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return db.CreateCollection(ctx, collectionName, nil)
	}
	return db.Collection(ctx, collectionName)
}
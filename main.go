package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kmrhemant916/prometheus-broadcaster/config"
	"github.com/kmrhemant916/prometheus-broadcaster/routes"
	"github.com/kmrhemant916/prometheus-broadcaster/utils"
	"github.com/nats-io/nats.go"
)

func main() {
	var config config.Config
	c, err:= config.ReadConf(os.Getenv("CONFIG_PATH"))
    if err != nil {
        panic(err)
    }
	nc, err := nats.Connect(os.Getenv("NATS_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	conn, err := utils.Connection(c.ArangoDB.Host, c.ArangoDB.Database, c.ArangoDB.Password, c.ArangoDB.Username)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	collection := utils.InitialiseServices(ctx, conn, c.ArangoDB.Database)
	r := routes.SetupRoutes(nc, conn, c, collection)
	go func() {
		sub, err := nc.SubscribeSync("alerts")
		if err != nil {
			log.Fatal(err)
		}
		for {
			start := time.Now()
			msg, err := sub.NextMsg(5 * time.Second)
			if err != nil {
				elapsed := time.Since(start)
				if elapsed >= 5*time.Second {
					log.Println("No message received")
					continue
				}
				log.Println("Error getting message:", err)
				continue
			}
			fmt.Println(string(msg.Data))
			// err = utils.SendEmail(string(msg.Data), c)
			// if err != nil {
			// 	log.Println("Error sending email:", err)
			// } else {
			// 	log.Println("Email sent successfully")
			// }
		}
	}()
	http.ListenAndServe(":"+c.Service.Port, r)
}
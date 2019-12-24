package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	_ "github.com/lib/pq"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {

	conn, err := rabbitmq.Dial("amqp://user:password@192.168.1.1:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"mydbname", // name
		false,      // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		db, err := sql.Open("postgres", `dbname= dbname user=myuser password=pass host=192.168.1.2 port=5432 sslmode=disable`)
		defer db.Close()
		failOnError(err, "Failed to declare a queue")
		for d := range msgs {
			str := bytes.Trim(d.Body, "\x00")
			log.Printf("Received a message: %s", str)
			result := strings.Replace(string(str), "E ", "", -1)
			_, err := db.Exec(string(result))
			if err != nil {
				fmt.Println("E----->", err)
			}

		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

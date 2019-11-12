
# pgsql2rmq

Postgres Fake Server is an implementation of a fake PostgreSQL server for forwarding execution instructions and creating virtual tables for queries by templates.

# Build and run

```
go build pgsql2rmq.go

./pgsql2rmq

```

# Example client from pgsql2rmq

```go
package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	db, err := sql.Open("postgres", `dbname= test user=user password=pass host=192.168.1.1 port=5432 sslmode=disable`)

	defer db.Close()

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"NewDB", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
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
		for d := range msgs {
			str := bytes.Trim(d.Body, "\x00")
			log.Printf("Received a message: %s", str)
			_, err := db.Exec(string(str))
			if err != nil {
				fmt.Println("E----->", err)
				//log.Fatalf("pg error:", err)
			}

		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
```
 



![Image alt](https://upload.wikimedia.org/wikipedia/commons/a/a0/Syrischer_Maler_von_1354_001.jpg)
Contact
-------
* Developer: Alexander Komarov <ignusius@gmail.com>

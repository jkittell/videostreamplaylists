package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"testing"
	"time"
)

func TestDeployment(t *testing.T) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"q.playlists", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	failOnError(err, "Failed to declare a queue")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job := Job{
		Id:  uuid.New(),
		URL: os.Getenv("URL"),
	}
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(job); err != nil {
		log.Println(err)
		return
	}
	err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         buffer.Bytes(),
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s", job.URL)
}

func TestGetPlaylistURLs(t *testing.T) {
	playlists := GetPlaylistURLs(os.Getenv("URL"))
	for i := 0; i < playlists.Length(); i++ {
		p := playlists.Lookup(i)
		log.Println(p)
	}
}

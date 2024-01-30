package main

import (
	"context"
	"github.com/google/uuid"
	"github.com/jkittell/data/database"
	"github.com/jkittell/data/structures"
	"log"
)

type Job struct {
	Id  uuid.UUID
	URL string
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	done := make(chan bool)
	jobs := make(chan Job)
	db, err := database.NewMongoDB[Playlist]()
	if err != nil {
		log.Fatal(err)
	}

	go receive(jobs)
	go track(db, jobs)
	<-done
}

func track(db database.MongoDB[Playlist], jobs chan Job) {
	tracking := structures.NewArray[string]()
	for job := range jobs {
		for i := 0; i < tracking.Length(); i++ {
			URL := tracking.Lookup(i)
			if job.URL == URL {
				continue
			}
		}
		if validate(job.URL) {
			go trackLivePlaylist(context.TODO(), db, job.URL)
			tracking.Push(job.URL)
		}
	}
}

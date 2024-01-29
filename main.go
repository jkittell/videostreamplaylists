package main

import (
	"context"
	"github.com/gofrs/uuid"
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
	jobs := make(chan Job)
	tracking := structures.NewArray[string]()

	ctx := context.Background()
	db, err := database.NewMongoDB[Playlist]()
	if err != nil {
		log.Fatal(err)
	}

	go receive(jobs)
	for job := range jobs {
		for i := 0; i < tracking.Length(); i++ {
			URL := tracking.Lookup(i)
			if job.URL == URL {
				continue
			} else {
				if validate(job.URL) {
					go trackLivePlaylist(ctx, db, job.URL)
					tracking.Push(job.URL)
				}
			}
		}
	}
}

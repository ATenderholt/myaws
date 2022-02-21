package main

import (
	"context"
	"myaws/database"
	"myaws/docker"
	"myaws/http"
	"myaws/lambda"
	"myaws/log"
	"myaws/s3"
	"myaws/sqs"
	"os"
	"os/signal"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s := <-c
		log.Info("Received signal %v", s)
		cancel()
	}()

	if err := start(ctx); err != nil {
		log.Error("Failed to start: %v", err)
	}
}

func start(ctx context.Context) error {
	log.Info("Starting up ...")

	initializeDb()
	initializeDocker()
	server, err := http.Serve()
	if err != nil {
		panic(err)
	}

	<-ctx.Done()

	log.Info("Shutting down ...")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer func() {
		cancel()
	}()

	err = server.Shutdown(ctxShutDown)
	if err != nil {
		log.Error("Error when shutting down HTTP server")
	}

	err = docker.ShutdownAll()
	if err != nil {
		log.Error("Errors when shutting down docker containers: %v", err)
	}

	return nil
}

func initializeDb() {
	var migrations database.Migrations
	migrations.AddAll(lambda.Migrations)

	log.Info("Initializing DB with %d Migrations.", migrations.Size())
	database.Initialize(migrations)
}

func initializeDocker() {
	docker.EnsureImage(s3.Image)
	err := docker.Start(s3.Container)
	if err != nil {
		panic(err)
	}

	docker.EnsureImage(sqs.Image)
	err = docker.Start(sqs.Container)
	if err != nil {
		panic(err)
	}
}

package main

import (
	"context"
	"myaws/database"
	"myaws/docker"
	"myaws/http"
	"myaws/lambda"
	"myaws/lambda/queries"
	"myaws/log"
	"myaws/moto"
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
	initializeDocker(ctx)
	server, err := http.Serve()
	if err != nil {
		panic(err)
	}

	<-ctx.Done()

	log.Info("Shutting down ...")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer func() {
		cancel()
	}()

	err = server.Shutdown(ctxShutDown)
	if err != nil {
		log.Error("Error when shutting down HTTP server")
	}

	err = docker.ShutdownAll(ctxShutDown)
	if err != nil {
		log.Error("Errors when shutting down docker containers: %v", err)
	}

	return nil
}

func initializeDb() {
	var migrations database.Migrations
	migrations.AddAll(lambda.Migrations)
	migrations.AddAll(moto.Migrations)
	migrations.AddAll(sqs.Migrations)

	log.Info("Initializing DB with %d Migrations.", migrations.Size())
	database.Initialize(migrations)
}

func initializeDocker(ctx context.Context) {
	// start moto first so a few seconds pass before trying to replay its events - potentially fragile!
	docker.EnsureImage(ctx, moto.Image)
	err := docker.Start(ctx, moto.Container)
	if err != nil {
		panic(err)
	}

	docker.EnsureImage(ctx, s3.Image)
	err = docker.Start(ctx, s3.Container)
	if err != nil {
		panic(err)
	}

	docker.EnsureImage(ctx, sqs.Image)
	err = docker.Start(ctx, sqs.Container)
	if err != nil {
		panic(err)
	}

	err = moto.ReplayAllToMoto(ctx)
	if err != nil {
		panic(err)
	}

	docker.EnsureImage(ctx, "mlupin/docker-lambda:python3.8")

	db := database.CreateConnection()
	functions, err := queries.LatestFunctions(ctx, db)
	if err != nil {
		panic(err)
	}

	for _, function := range functions {
		err := lambda.StartFunction(ctx, &function)
		if err != nil {
			panic(err)
		}
	}
}

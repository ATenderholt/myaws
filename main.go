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
	"myaws/settings"
	"myaws/sqs"
	"os"
	"os/signal"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cfg := settings.DefaultConfig()
	mainCtx := cfg.NewContext(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(mainCtx)
	go func() {
		s := <-c
		log.Info("Received signal %v", s)
		cancel()
	}()

	if err := start(ctx, cfg); err != nil {
		log.Error("Failed to start: %v", err)
	}
}

func start(ctx context.Context, config *settings.Config) error {
	log.Info("Starting up ...")

	initializeDb(config)
	initializeDocker(ctx, config)
	server, err := http.Serve(config)
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

func initializeDb(cfg *settings.Config) {
	var migrations database.Migrations
	migrations.AddAll(lambda.Migrations)
	migrations.AddAll(moto.Migrations)
	migrations.AddAll(sqs.Migrations)

	log.Info("Initializing DB with %d Migrations.", migrations.Size())
	database.Initialize(cfg, migrations)
}

func initializeDocker(ctx context.Context, cfg *settings.Config) {
	db := database.CreateConnection(cfg)

	go initializeMoto(ctx)
	go initializeElasticMQ(ctx, db)

	docker.EnsureImage(ctx, s3.Image)
	_, err := docker.Start(ctx, s3.Container, "")
	if err != nil {
		panic(err)
	}

	docker.EnsureImage(ctx, "mlupin/docker-lambda:python3.8")

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

func initializeMoto(ctx context.Context) {
	docker.EnsureImage(ctx, moto.Image)
	motoReady, err := docker.Start(ctx, moto.Container, "Running on http")
	if err != nil {
		panic(err)
	}

	<-motoReady

	log.Info("Moto is ready, starting replay ...")

	err = moto.ReplayAllToMoto(ctx)
	if err != nil {
		panic(err)
	}
}

func initializeElasticMQ(ctx context.Context, db *database.Database) {
	docker.EnsureImage(ctx, sqs.Image)
	elasticReady, err := docker.Start(ctx, sqs.Container, "started in")
	if err != nil {
		panic(err)
	}

	<-elasticReady

	log.Info("ElasticMQ is ready, starting event sources ...")

	eventSource, err := queries.LoadEventSource(ctx, db, "0486b330-6eed-48eb-87ac-742ab978db18")
	if err != nil {
		panic(err)
	}

	err = lambda.StartEventSource(ctx, eventSource)
	if err != nil {
		panic(err)
	}
}

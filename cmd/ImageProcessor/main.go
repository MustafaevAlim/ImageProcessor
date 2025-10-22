package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	"ImageProcessor/internal/api"
	"ImageProcessor/internal/api/handlers"
	"ImageProcessor/internal/app"
	"ImageProcessor/internal/config"
	"ImageProcessor/internal/repository"
)

func main() {
	zlog.Init()
	cfg, err := config.NewConfig(".env")
	if err != nil {
		zlog.Logger.Fatal().Msg(err.Error())
	}

	pgDSN := fmt.Sprintf(
		"host=%s user=%s password=%s database=%s sslmode=disable",
		cfg.Postgre.Host,
		cfg.Postgre.User,
		cfg.Postgre.Password,
		cfg.Postgre.DBName,
	)
	db, err := repository.NewStorage(pgDSN)
	if err != nil {
		zlog.Logger.Fatal().Msg(err.Error())
	}

	engine := ginext.New("debug")

	consumer := repository.NewImageConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.GroupID,
	)

	producer := repository.NewImageProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
	)

	minio, err := repository.NewImageStorage(
		cfg.Minio.Endpoint,
		cfg.Minio.User,
		cfg.Minio.Password,
		cfg.Minio.BucketName,
		cfg.Minio.Sslmode,
	)
	if err != nil {
		zlog.Logger.Fatal().Msg(err.Error())
	}

	h := handlers.NewHandler(db, producer, minio)
	api.SetupRoutes(h, engine)

	a := app.App{
		DB:           db,
		Consumer:     consumer,
		Producer:     producer,
		Handler:      engine,
		Host:         ":" + cfg.Server.Port,
		ImageStorage: minio,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	if err := a.Run(ctx); err != nil {
		zlog.Logger.Fatal().Msg(err.Error())
	}

}

package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	"ImageProcessor/internal/model"
	"ImageProcessor/internal/repository"
	"ImageProcessor/internal/service"
)

type App struct {
	Host         string
	Handler      *ginext.Engine
	DB           repository.Storager
	Consumer     repository.ImageTaskConsumer
	Producer     repository.ImageTaskProducer
	ImageStorage repository.ImageStore
}

func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	serv := http.Server{
		Addr:    a.Host,
		Handler: a.Handler,
	}

	imgTaskCh := make(chan kafka.Message, 100)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := serv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zlog.Logger.Error().Msgf("Server listen error: %s", err.Error())
			cancel()
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		go a.Consumer.ConsumeTask(ctx, imgTaskCh)
	}()

	wg.Add(1)
	go func() {
		is := service.ImageService{
			Ctx:          ctx,
			ImageStorage: a.ImageStorage,
		}
		defer wg.Done()
		var img model.ImageTask
		for msg := range imgTaskCh {
			err := json.Unmarshal(msg.Value, &img)
			if err != nil {
				zlog.Logger.Error().Msgf("Unmarshal image error: %s", err.Error())
				continue
			}
			is.Img = img
			updateImg, err := service.ProcessImage(is)
			if err != nil {
				zlog.Logger.Error().Msgf("Process image error: %s", err.Error())
				continue
			}

			err = a.DB.UpdateImage(ctx, updateImg)
			if err != nil {
				zlog.Logger.Error().Msgf("Commit message error: %s", err.Error())
				continue
			}

			err = a.Consumer.CommitOffset(ctx, msg)
			if err != nil {
				zlog.Logger.Error().Msgf("Commit message error: %s", err.Error())
				continue
			}
		}
	}()

	<-ctx.Done()

	zlog.Logger.Info().Msg("App shutdown...")

	err := serv.Shutdown(ctx)
	if err != nil {
		zlog.Logger.Error().Msgf("Server shutdown error: %s", err.Error())
	}

	wg.Wait()

	err = a.DB.Close()
	if err != nil {
		zlog.Logger.Error().Msgf("DB close error: %v", err)
	}

	err = a.Consumer.Close()
	if err != nil {
		zlog.Logger.Error().Msgf("Consumer close error: %v", err)
	}

	err = a.Producer.Close()
	if err != nil {
		zlog.Logger.Error().Msgf("Producer close error: %v", err)
	}

	return nil
}

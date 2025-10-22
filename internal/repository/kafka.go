package repository

import (
	"context"
	"encoding/json"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"ImageProcessor/internal/model"
)

type ImageTaskProducer interface {
	Publish(ctx context.Context, imgTask model.ImageTask) error
	Close() error
}

type ImageTaskConsumer interface {
	ConsumeTask(ctx context.Context, out chan<- kafkago.Message)
	CommitOffset(ctx context.Context, img kafkago.Message) error
	Close() error
}

type ImageConsumer struct {
	Consumer *kafka.Consumer
}

type ImageProducer struct {
	Producer *kafka.Producer
}

func NewImageConsumer(brokers []string, topic, groupID string) *ImageConsumer {
	c := kafka.NewConsumer(brokers, topic, groupID)
	return &ImageConsumer{Consumer: c}
}

func (c *ImageConsumer) ConsumeTask(ctx context.Context, out chan<- kafkago.Message) {

	defer close(out)
	for {

		msg, err := c.Consumer.FetchWithRetry(ctx, retry.Strategy{
			Attempts: 1,
			Delay:    1 * time.Second,
			Backoff:  2,
		})
		if err != nil {
			if errCtx := ctx.Err(); errCtx != nil {
				return
			}
			zlog.Logger.Error().Msgf("Consumer fetch error: %s", err.Error())
			continue
		}

		select {
		case out <- msg:
		case <-ctx.Done():
			return
		}

	}

}

func (c *ImageConsumer) CommitOffset(ctx context.Context, msg kafkago.Message) error {
	return c.Consumer.Commit(ctx, msg)
}

func (c *ImageConsumer) Close() error {
	return c.Consumer.Close()
}

func NewImageProducer(brokers []string, topic string) *ImageProducer {
	p := kafka.NewProducer(brokers, topic)
	return &ImageProducer{Producer: p}
}

func (p *ImageProducer) Publish(ctx context.Context, imgTask model.ImageTask) error {
	data, err := json.Marshal(imgTask)
	if err != nil {
		return err
	}

	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    1 * time.Second,
		Backoff:  2,
	}

	return p.Producer.SendWithRetry(ctx, strategy, nil, data)
}

func (p *ImageProducer) Close() error {
	return p.Producer.Close()
}

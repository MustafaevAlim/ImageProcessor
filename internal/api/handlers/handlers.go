package handlers

import (
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	"ImageProcessor/internal/repository"
)

type Handler struct {
	DB           repository.Storager
	Producer     repository.ImageTaskProducer
	ImageStorage repository.ImageStore
}

func NewHandler(db repository.Storager, p repository.ImageTaskProducer, i repository.ImageStore) *Handler {
	return &Handler{DB: db, Producer: p, ImageStorage: i}
}

func WriteJSONError(c *ginext.Context, err error, status int) {
	zlog.Logger.Error().Msg(err.Error())
	c.JSON(status, ginext.H{
		"error": err.Error(),
	})
}

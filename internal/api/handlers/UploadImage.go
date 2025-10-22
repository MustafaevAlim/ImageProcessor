package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	"ImageProcessor/internal/model"
)

func (h *Handler) UploadImage(c *ginext.Context) {
	fileHeader, err := c.FormFile("img")
	if err != nil {
		WriteJSONError(c, err, http.StatusBadRequest)
		return
	}

	ext := filepath.Ext(fileHeader.Filename)
	if !slices.Contains([]string{".png", ".gif", ".jpeg", ".jpg"}, ext) {
		WriteJSONError(c, fmt.Errorf("unsupported format"), http.StatusBadRequest)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		WriteJSONError(c, err, http.StatusBadRequest)
		return
	}
	defer file.Close()

	objectName := fmt.Sprintf("uploads/%s%s", uuid.New().String(), ext)

	err = h.ImageStorage.Upload(context.Background(), file, objectName, fileHeader.Size)
	if err != nil {
		WriteJSONError(c, err, http.StatusInternalServerError)
		return
	}

	typeProcessing := c.PostForm("type_processing")

	task := model.ImageTask{
		TypeProcessing: typeProcessing,
		UploadsPath:    objectName,
	}

	err = getParameters(c, typeProcessing, &task, h)
	if err != nil {
		WriteJSONError(c, err, http.StatusBadRequest)
		return
	}

	img := model.ImageInCreate{
		UploadsPath: objectName,
	}

	id, err := h.DB.CreateImage(c.Request.Context(), img)
	if err != nil {
		WriteJSONError(c, err, http.StatusInternalServerError)
		return
	}
	task.ImageID = id

	err = h.Producer.Publish(context.Background(), task)
	if err != nil {
		WriteJSONError(c, err, http.StatusInternalServerError)
		return
	}
	zlog.Logger.Info().Msg("Message publish")

	c.JSON(http.StatusOK, ginext.H{
		"result":      "image publish in queue",
		"object_name": objectName,
		"image_id":    id,
	})
}

func getHeigthAndWidth(c *ginext.Context) (int, int, error) {
	heightStr := c.PostForm("height")
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		return 0, 0, err
	}

	widthStr := c.PostForm("width")
	width, err := strconv.Atoi(widthStr)
	if err != nil {
		return 0, 0, err
	}
	return height, width, nil
}

func getParameters(c *ginext.Context, typeProcessing string, task *model.ImageTask, h *Handler) error {
	switch typeProcessing {
	case "resize":
		height, width, err := getHeigthAndWidth(c)
		if err != nil {
			return err
		}
		task.Parameters.Height = &height
		task.Parameters.Width = &width

	case "watermark":
		fileHeader, err := c.FormFile("watermark")
		if err != nil {
			return err
		}

		file, err := fileHeader.Open()
		if err != nil {
			return err
		}
		defer file.Close()

		ext := filepath.Ext(fileHeader.Filename)
		watermarkObjectName := fmt.Sprintf("watermarks/%s%s", uuid.New().String(), ext)

		err = h.ImageStorage.Upload(context.Background(), file, watermarkObjectName, fileHeader.Size)
		if err != nil {
			return err
		}

		task.Parameters.WatermarkPath = &watermarkObjectName
	}
	return nil
}

package handlers

import (
	"ImageProcessor/internal/model"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/wb-go/wbf/ginext"
)

func (h *Handler) GetImages(c *ginext.Context) {
	lastCreatedAtStr := c.Query("last_created_at")
	lastCreatedAt, err := time.Parse(time.RFC3339, lastCreatedAtStr)
	if err != nil {
		WriteJSONError(c, err, http.StatusBadRequest)
		return
	}

	lastIDStr := c.DefaultQuery("last_id", "0")
	lastID, err := strconv.Atoi(lastIDStr)
	if err != nil {
		WriteJSONError(c, err, http.StatusBadRequest)
		return
	}

	mode := c.Query("mode")
	images, err := h.DB.GetImages(context.Background(), lastCreatedAt, lastID, mode)
	if err != nil {
		WriteJSONError(c, err, http.StatusInternalServerError)
		return
	}

	count, err := h.DB.GetCountImages(context.Background())
	if err != nil {
		WriteJSONError(c, err, http.StatusInternalServerError)
		return
	}

	var imageWithUrl []struct {
		model.ImageInRepo
		Url string `json:"url"`
	}
	var url string
	for _, v := range images {
		if v.Processed {
			url = v.ProcessedPath
		} else {
			url = v.UploadsPath
		}
		imageWithUrl = append(imageWithUrl, struct {
			model.ImageInRepo
			Url string `json:"url"`
		}{v, "/images/" + url})
	}

	c.JSON(http.StatusOK, ginext.H{
		"count":  count,
		"images": imageWithUrl,
	})
}

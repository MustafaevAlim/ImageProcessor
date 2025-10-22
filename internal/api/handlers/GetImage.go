package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/wb-go/wbf/ginext"
)

func (h *Handler) GetImage(c *ginext.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		WriteJSONError(c, err, http.StatusBadRequest)
		return
	}

	img, err := h.DB.GetImage(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, ginext.H{
				"error": "not found",
			})
			return
		}
		WriteJSONError(c, err, http.StatusInternalServerError)
		return
	}

	if !img.Processed {
		WriteJSONError(c, fmt.Errorf("image processing"), http.StatusAccepted)
		return
	}

	var url string
	if img.Processed {
		url = img.ProcessedPath
	} else {
		url = img.UploadsPath
	}

	c.JSON(http.StatusOK, ginext.H{
		"url": "/images/" + url,
	})

}

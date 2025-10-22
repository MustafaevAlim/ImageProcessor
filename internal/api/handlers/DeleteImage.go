package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/wb-go/wbf/ginext"
)

func (h *Handler) DeleteImage(c *ginext.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		WriteJSONError(c, err, http.StatusBadRequest)
		return
	}

	err = h.DB.DeleteImage(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			WriteJSONError(c, fmt.Errorf("not found"), http.StatusNotFound)
			return
		}
		WriteJSONError(c, err, http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, ginext.H{
		"result": "image delete",
	})
}

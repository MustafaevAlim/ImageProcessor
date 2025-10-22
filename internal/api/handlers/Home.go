package handlers

import (
	"net/http"

	"github.com/wb-go/wbf/ginext"
)

func (h *Handler) Home(c *ginext.Context) {
	c.HTML(http.StatusOK, "index.html", ginext.H{"result": "ok"})
}

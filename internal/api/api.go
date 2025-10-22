package api

import (
	"github.com/wb-go/wbf/ginext"

	"ImageProcessor/internal/api/handlers"
)

func SetupRoutes(h *handlers.Handler, g *ginext.Engine) {
	g.Use(ginext.Logger(), ginext.Recovery())
	g.LoadHTMLGlob("web/*.html")

	g.POST("/upload", h.UploadImage)
	g.GET("/image/:id", h.GetImage)
	g.GET("/images", h.GetImages)
	g.DELETE("/image/:id", h.DeleteImage)
	g.GET("/", h.Home)
}

package main

import (
	"context"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func registerRoutes(h *server.Hertz) {
	h.GET("/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, utils.H{"status": "ok", "service": "quevia-api"})
	})
}

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	h := server.Default(server.WithHostPorts(":" + port))
	registerRoutes(h)
	h.Spin()
}

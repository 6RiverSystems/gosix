package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go.6river.tech/gosix/registry"
	"go.6river.tech/gosix/server"
)

type KillController struct {
}

func (k *KillController) Register(_ *registry.Registry, router gin.IRouter) error {
	rg := router.Group("/server")
	rg.POST("/kill", k.HandleKill)
	rg.POST("/shutdown", k.HandleShutdown)
	return nil
}

func (k *KillController) HandleKill(c *gin.Context) {
	svr, services := shutdownHelpers(c)
	c.String(http.StatusOK, "Goodbye cruel world!\n")

	// do the shutdown in the background
	go server.Shutdown(svr, services, false)
}

func (k *KillController) HandleShutdown(c *gin.Context) {
	svr, services := shutdownHelpers(c)
	c.String(http.StatusOK, "Daisy, Daisy, Give me your answer, do!\n")

	// do the shutdown in the background
	go server.Shutdown(svr, services, true)
}

func shutdownHelpers(c *gin.Context) (*http.Server, *registry.Registry) {
	s := server.HttpServer(c)
	vs := server.Values(c)
	reg, _ := registry.GetRegistry(vs)
	return s, reg
}

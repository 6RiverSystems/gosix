// Copyright (c) 2021 6 River Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go.6river.tech/gosix/registry"
	"go.6river.tech/gosix/server"
)

type KillController struct{}

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

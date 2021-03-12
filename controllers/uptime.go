package controllers

import (
	"net/http"
	"time"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"

	"go.6river.tech/gosix/logging"
	"go.6river.tech/gosix/oas"
	"go.6river.tech/gosix/registry"
)

type UptimeController struct {
	startTime oas.Time
	logger    *logging.Logger
	registry  *registry.Registry
}

func (u *UptimeController) initialize(reg *registry.Registry) {
	if u.startTime == (oas.Time{}) {
		u.startTime = oas.Now()
	}
	if u.logger == nil {
		u.logger = logging.GetLogger("controllers/uptime")
	}
	if u.registry == nil {
		u.registry = reg
	} else if u.registry != reg {
		panic("Uptime controller doesn't support multiple registries")
	}
}

func (u *UptimeController) Register(reg *registry.Registry, router gin.IRouter) error {
	u.initialize(reg)
	router.GET("/", u.Handle)
	// a slow variant for testing things like graceful shutdown
	router.GET("/slow", u.HandleSlow)
	return nil
}

func (u *UptimeController) Handle(c *gin.Context) {
	// don't reply until services are all started
	if u.registry != nil {
		if err := u.registry.WaitAllReady(c.Request.Context()); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err) // nolint:errcheck
			return
		}
	}
	l := location.Get(c)
	c.AsciiJSON(http.StatusOK, gin.H{
		"startTime": u.startTime,
		"location":  l,
		// FIXME: report app version via registry.Values injection
	})
}

type slowParams struct {
	Delay int `form:"delay" binding:"required"`
}

func (u *UptimeController) HandleSlow(c *gin.Context) {
	var p slowParams
	if c.Bind(&p) != nil {
		c.String(http.StatusBadRequest, "Must provide 'delay' parameter for slow request")
		return
	}
	u.logger.Info().Msg("starting slow request")
	time.Sleep(time.Duration(p.Delay) * time.Millisecond)
	u.logger.Info().Msg("finishing slow request")
	u.Handle(c)
}

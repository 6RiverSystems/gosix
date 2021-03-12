package server

import (
	_ "net/http/pprof"
	"os"

	"github.com/Depado/ginprom"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/location"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"go.6river.tech/gosix/logging"

	// gin-gonic's cors module is a mess
	cors "github.com/rs/cors/wrapper/gin"
)

func NewEngine() *gin.Engine {
	if os.Getenv("GIN_MODE") == "" {
		// TODO: don't use NODE_ENV for things that aren't nodejs
		switch os.Getenv("NODE_ENV") {
		case "production":
			gin.SetMode(gin.ReleaseMode)
		case "development", "":
			gin.SetMode(gin.DebugMode)
		case "test", "acceptance":
			// TODO: actually probably want debug mode here too?
			gin.SetMode(gin.TestMode)
		default:
			panic(errors.Errorf("Unrecognized NODE_ENV value: '%s'", os.Getenv("NODE_ENV")))
		}
	}

	// logging is all going through zerolog, which won't support color
	gin.DisableConsoleColor()

	ginDebugLogger := logging.GetLogger("gin/debug")
	gin.DefaultWriter = ginDebugLogger
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, numHandlers int) {
		ginDebugLogger.Debug().
			Str("method", httpMethod).
			Str("path", absolutePath).
			Str("handler", handlerName).
			Int("numHandlers", numHandlers).
			Msg("route added")
	}

	r := gin.New()

	// attach custom logging
	// FIXME: this logger won't see level changes, would need to make our own lib
	requestLogger := logging.GetLogger("gin/request").Current()
	r.Use(logger.SetLogger(logger.Config{
		Logger: &requestLogger,
		UTC:    true,
	}))

	// Set up prometheus, before recovery so it can see the recovered responses
	p := ginprom.New(
		ginprom.Engine(r),
		ginprom.Path("/metrics"),
	)
	r.Use(p.Instrument())

	// gzip encoding
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithDecompressFn(gzip.DefaultDecompressHandle)))

	// attach recovery middleware using request logger
	// This needs to be AFTER the gzip layer, as the gzip layer has a defer that
	// will result in sending an empty 200 during panic recovery if this is above
	// that layer
	// TODO: we should make a better recovery handler that is better suited to
	// our idioms and to zerolog, including not trying to color things even when
	// console coloring above is disabled
	r.Use(gin.RecoveryWithWriter(requestLogger))

	// basic CORS policy
	// debugging note: unlike loopback(4), this cors extension will only send the Accept-* header(s)
	// when the request includes the Origin header. This is fine.
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}))

	// make info from reverse proxy headers available to handlers
	r.Use(location.Default())

	return r
}

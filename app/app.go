// package app provides a helper for building the common boilerplate for apps
// using this library.
package app

import (
	"context"
	"expvar"
	"net/http"
	"os"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/getkin/kin-openapi/openapi3"
	ginexpvar "github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"google.golang.org/grpc"

	"go.6river.tech/gosix/controllers"
	"go.6river.tech/gosix/db"
	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/ginmiddleware"
	grpccommon "go.6river.tech/gosix/grpc"
	"go.6river.tech/gosix/logging"
	"go.6river.tech/gosix/migrate"
	"go.6river.tech/gosix/pubsub"
	"go.6river.tech/gosix/registry"
	"go.6river.tech/gosix/server"
	swaggerui "go.6river.tech/gosix/swagger-ui"
)

type App struct {
	// Name is the primary name of the application. It will be used to provide
	// default configuration for several elements, including logging and database
	// access.
	Name string

	// Version is the version of the app that is running. Typically this will come
	// from build-time generated code.
	Version string

	// InitDbMigration provides a hook for loading the proper db migration scripts
	// from the app. The shared code will handle running the migrations, but this
	// hook is responsible for any sorting that is needed as part of registering
	// them with the Migrator.
	InitDbMigration func(ctx context.Context, m *migrate.Migrator) error

	InitEnt func(ctx context.Context, drv *sql.Driver, logger func(args ...interface{}), debug bool) (ent.EntClient, error)

	// LoadOASSpec provides a hook for enabling OAS validation middleware, using
	// the returned swagger spec.
	LoadOASSpec func(ctx context.Context) (*openapi3.T, error)

	// OASFS specifies a "filesystem" to use for the `/oas` route, to serve
	// swagger specs to the swagger-ui (and thus this field enables the swagger-ui
	// routes too). This is typically an embed.FS wrapped in http.FS.
	OASFS http.FileSystem

	// SwaggerUIConfigHandler provides an optional override for handling the
	// `/oas-ui-config` route, for when extra OAS specs should be included, such
	// as when the gRPC HTTP gateway is in use. Note that the swagger ui (and thus
	// this) are only used if OASFS is non-nil.
	SwaggerUIConfigHandler http.HandlerFunc

	// Port is the default base listening port for the app to use. Debug/Test
	// configurations will commonly offset from this port. HTTP will typically run
	// on the base port (after any debug/test offsets), others will run offset
	// from this.
	Port int

	WithPubsubClient bool

	Grpc *AppGrpc

	Registry         *registry.Registry
	RegisterServices func(context.Context, *registry.Registry, registry.MutableValues) error
	CustomizeRoutes  func(context.Context, *gin.Engine, *registry.Registry, registry.MutableValues) error

	// App "ISA" DI root
	registry.MutableValues
}

type AppGrpc struct {
	PortOffset     int
	ServerOpts     []grpc.ServerOption
	Initializer    grpccommon.GrpcInitializer
	GatewayPaths   []string
	OnGatewayStart func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error
}

func (app *App) WithDefaults() *App {
	if app.MutableValues == nil {
		app.MutableValues = registry.NewValues()
	}
	if app.Registry == nil {
		app.Registry = registry.New(app.Name)
		app.Bind(registry.RegistryKey, registry.ConstantValue(app.Registry))
	}
	return app
}

var entClientKey = registry.InterfaceAt("ent-client", (*ent.EntClient)(nil))
var pubsubClientKey = registry.InterfaceAt("pubsub-client", (*pubsub.Client)(nil))

func (app *App) Main() (err error) {
	logging.ConfigureDefaultLogging()
	logger := logging.GetLogger(app.Name)

	db.SetDefaultDbName(app.Name)

	if app.Port == 0 {
		server.EnableRandomPorts()
	}

	// report the app version as an expvar
	expvar.NewString("version/" + app.Name).Set(app.Version)

	// setup default prometheus metrics
	prometheus.DefaultRegisterer.MustRegister(collectors.NewBuildInfoCollector())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// TODO: should we expose a cached view instead here?
	ctx = registry.WithValues(ctx, app.MutableValues)

	var drv *sql.Driver
	drv, err = app.openDB(ctx, logger)
	if drv != nil {
		defer func() {
			if err := drv.Close(); err != nil {
				logger.Error().Err(err).Msg("Failed to cleanup SQL connection")
			}
		}()
	}
	if err != nil {
		return err
	}

	var client ent.EntClient
	// client.Close() just is a wrapper around drv.Close(), so we don't need to
	// setup a separate defer for it
	if client, err = app.setupDB(ctx, logger, drv); err != nil {
		return err
	}
	app.Bind(entClientKey, registry.ConstantValue(client))

	if app.WithPubsubClient {
		// app name needs to be sanitized for prometheus
		// TODO: this is not nearly complete, just enough to catch simple problems
		promNs := strings.ReplaceAll(app.Name, "-", "_")
		if psc, err := pubsub.DefaultClient(promNs); err != nil {
			return err
		} else {
			app.Bind(pubsubClientKey, registry.ConstantValue(psc))
		}
	}

	var engine *gin.Engine
	if engine, err = app.setupGin(ctx, client); err != nil {
		return err
	}

	if err = app.setupGrpc(ctx, engine); err != nil {
		return err
	}

	registry.RegisterDefaultSignalListener(app.Registry)

	if app.RegisterServices != nil {
		if err := app.RegisterServices(ctx, app.Registry, app.MutableValues); err != nil {
			return err
		}
	}

	return app.Registry.RunDefault(ctx, client, logger)
}

func (app *App) EntClient() (ent.EntClient, bool) {
	c, ok := app.Value(entClientKey)
	return c.(ent.EntClient), ok
}

func (app *App) PubsubClient() (pubsub.Client, bool) {
	c, ok := app.Value(pubsubClientKey)
	return c.(pubsub.Client), ok
}

func (app *App) openDB(ctx context.Context, logger *logging.Logger) (drv *sql.Driver, err error) {
	// for 6mon friendliness, wait for up to 120 seconds for a db connection
	if err = func() error {
		dbWaitCtx, dbWaitCancel := context.WithDeadline(ctx, time.Now().Add(120*time.Second))
		defer dbWaitCancel()
		return db.WaitForDB(dbWaitCtx)
	}(); err != nil {
		return nil, err
	}

	if drv, err = db.OpenSqlForEnt(); err != nil {
		if drv != nil {
			if err := drv.Close(); err != nil {
				logger.Error().Err(err).Msg("Failed to cleanup SQL connection")
			}
		}
		return nil, err
	}
	return drv, nil
}

func (app *App) setupDB(ctx context.Context, logger *logging.Logger, drv *sql.Driver) (client ent.EntClient, err error) {
	sqlLogger := logging.GetLogger(app.Name + "/sql")
	sqlLoggerFunc := func(args ...interface{}) {
		for _, m := range args {
			// currently ent only sends us strings
			sqlLogger.Trace().Msg(m.(string))
		}
	}
	entDebug := strings.Contains(os.Getenv("DEBUG"), "sql")
	if client, err = app.InitEnt(ctx, drv, sqlLoggerFunc, entDebug); err != nil {
		return client, err
	}
	// Setup db prometheus metrics
	prometheus.DefaultRegisterer.MustRegister(collectors.NewDBStatsCollector(drv.DB(), db.GetDefaultDbName()))

	m := &migrate.Migrator{}
	if app.InitDbMigration != nil {
		if err = app.InitDbMigration(ctx, m); err != nil {
			return client, err
		}
	}
	if err = db.Up(ctx, client, m); err != nil {
		return client, err
	}

	return client, nil
}

func (app *App) setupGin(ctx context.Context, client ent.EntClient) (*gin.Engine, error) {
	engine := server.NewEngine()
	engine.Use(ginmiddleware.WithEntClient(client, db.GetDefaultDbName()))

	// Enable `format: uuid` validation
	openapi3.DefineStringFormat("uuid", openapi3.FormatOfStringForUUIDOfRFC4122)

	if app.LoadOASSpec != nil {
		if spec, err := app.LoadOASSpec(ctx); err != nil {
			return nil, err
		} else if spec != nil {
			engine.Use(ginmiddleware.WithOASValidation(
				spec,
				true,
				ginmiddleware.AllowUndefinedRoutes(
					ginmiddleware.DefaultOASErrorHandler,
				),
				nil,
			))
		}
	}

	if app.OASFS != nil {
		// TODO: wildcard route doesn't permit this to overlap with the `/oas` "fs"
		// TODO: this won't work properly behind a path-modifying reverse proxy as we
		// don't have any `servers` entries so it will guess the wrong base
		engine.StaticFS("/oas-ui", http.FS(swaggerui.FS))
		configHandler := app.SwaggerUIConfigHandler
		if configHandler == nil {
			configHandler = swaggerui.DefaultConfigHandler()
		}
		type msi = map[string]interface{}
		engine.GET(swaggerui.ConfigLoadingPath, gin.WrapF(configHandler))
		// NOTE: this will serve yaml as text/plain. YAML doesn't have a standardized
		// mime type, so that's OK for now
		engine.StaticFS("/oas", app.OASFS)
	}

	// add standard debug routes
	engine.GET("/debug/vars", ginexpvar.Handler())
	// use a wildcard route and defert to the default servemux, so that
	// we don't have to replicate the Index wildcard behavior ourselves
	engine.Any("/debug/pprof/*profile", gin.WrapH(http.DefaultServeMux))
	controllers.AddCommonControllers(app.Registry)
	if app.CustomizeRoutes != nil {
		if err := app.CustomizeRoutes(ctx, engine, app.Registry, app.MutableValues); err != nil {
			return nil, err
		}
	}
	if err := app.Registry.RegisterControllers(engine); err != nil {
		return nil, err
	}

	server.RegisterHttpService(app.Registry, engine, app.Port, 0)
	return engine, nil
}

func (app *App) setupGrpc(_ context.Context, engine *gin.Engine) error {
	if app.Grpc != nil {
		grpcServiceTag := app.Registry.AddService(grpccommon.NewGrpcService(
			app.Port, app.Grpc.PortOffset,
			app.Grpc.ServerOpts,
			app.Grpc.Initializer,
		))
		app.Registry.AddService(grpccommon.NewGatewayService(
			app.Name,
			app.Port, app.Grpc.PortOffset,
			engine,
			app.Grpc.GatewayPaths,
			grpcServiceTag,
			app.Grpc.OnGatewayStart,
		))
	}
	return nil
}

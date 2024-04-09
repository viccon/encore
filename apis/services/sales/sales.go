// Package sales represent the encore application.
package sales

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"encore.dev"
	"encore.dev/rlog"
	"github.com/ardanlabs/conf/v3"
	"github.com/ardanlabs/encore/app/api/debug"
	"github.com/ardanlabs/encore/app/api/metrics"
	"github.com/ardanlabs/encore/app/core/crud/homeapp"
	"github.com/ardanlabs/encore/app/core/crud/productapp"
	"github.com/ardanlabs/encore/app/core/crud/tranapp"
	"github.com/ardanlabs/encore/app/core/crud/userapp"
	"github.com/ardanlabs/encore/app/core/views/vproductapp"
	"github.com/ardanlabs/encore/business/core/crud/delegate"
	"github.com/ardanlabs/encore/business/core/crud/homebus"
	"github.com/ardanlabs/encore/business/core/crud/homebus/stores/homedb"
	"github.com/ardanlabs/encore/business/core/crud/productbus"
	"github.com/ardanlabs/encore/business/core/crud/productbus/stores/productdb"
	"github.com/ardanlabs/encore/business/core/crud/userbus"
	"github.com/ardanlabs/encore/business/core/crud/userbus/stores/userdb"
	"github.com/ardanlabs/encore/business/core/views/vproductbus"
	"github.com/ardanlabs/encore/business/core/views/vproductbus/stores/vproductdb"
	"github.com/ardanlabs/encore/business/data/appdb"
	"github.com/ardanlabs/encore/business/data/appdb/migrate"
	"github.com/ardanlabs/encore/business/data/sqldb"
	"github.com/jmoiron/sqlx"
)

type appCrud struct {
	homeApp    *homeapp.Core
	productApp *productapp.Core
	tranApp    *tranapp.Core
	userApp    *userapp.Core
}

type appView struct {
	vproductApp *vproductapp.Core
}

type busCrud struct {
	homeBus    *homebus.Core
	productBus *productbus.Core
	userBus    *userbus.Core
}

// Service represents the encore service application.
//
//encore:service
type Service struct {
	log   rlog.Ctx
	mtrcs *metrics.Values
	db    *sqlx.DB
	debug http.Handler
	appCrud
	appView
	busCrud
}

// NewService is called to create a new encore Service.
func NewService(log rlog.Ctx, db *sqlx.DB) (*Service, error) {
	delegate := delegate.New(log)
	userBus := userbus.NewCore(delegate, userdb.NewStore(log, db))
	productBus := productbus.NewCore(log, userBus, delegate, productdb.NewStore(log, db))
	homeBus := homebus.NewCore(userBus, delegate, homedb.NewStore(log, db))
	vproductBus := vproductbus.NewCore(vproductdb.NewStore(log, db))

	s := Service{
		log:   log,
		mtrcs: newMetrics(),
		db:    db,
		debug: debug.Mux(),
		appCrud: appCrud{
			userApp:    userapp.NewCore(userBus),
			productApp: productapp.NewCore(productBus),
			homeApp:    homeapp.NewCore(homeBus),
			tranApp:    tranapp.NewCore(userBus, productBus),
		},
		appView: appView{
			vproductApp: vproductapp.NewCore(vproductBus),
		},
		busCrud: busCrud{
			userBus:    userBus,
			productBus: productBus,
			homeBus:    homeBus,
		},
	}

	return &s, nil
}

// Shutdown implements a function that will be called by encore when the service
// is signaled to shutdown.
func (s *Service) Shutdown(force context.Context) {
	defer s.log.Info("shutdown", "status", "shutdown complete")

	s.log.Info("shutdown", "status", "stopping database support")
	s.db.Close()
}

// =============================================================================

// initService is called by Encore to initialize the service.
//
//lint:ignore U1000 "called by encore"
func initService() (*Service, error) {
	log := rlog.With("service", "sales")

	db, err := startup(log)
	if err != nil {
		return nil, err
	}

	return NewService(log, db)
}

func startup(log rlog.Ctx) (*sqlx.DB, error) {

	// -------------------------------------------------------------------------
	// GOMAXPROCS

	log.Info("initService", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// -------------------------------------------------------------------------
	// Configuration

	cfg := struct {
		conf.Version
		DB struct {
			MaxIdleConns int `conf:"default:2"`
			MaxOpenConns int `conf:"default:0"`
		}
	}{
		Version: conf.Version{
			Build: encore.Meta().Environment.Name,
			Desc:  "Service Project",
		},
	}

	const prefix = "SALES"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil, err
		}
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Info("initService", "environment", encore.Meta().Environment.Name)

	out, err := conf.String(&cfg)
	if err != nil {
		return nil, fmt.Errorf("generating config for output: %w", err)
	}
	log.Info("initService", "config", out)

	// -------------------------------------------------------------------------
	// Database Support

	log.Info("initService", "status", "initializing database support")

	db, err := sqldb.Open(sqldb.Config{
		EDB:          appdb.AppDB,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxOpenConns: cfg.DB.MaxOpenConns,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	// TODO: I don't like this here because it's more of an ops thing, but
	// for now I will leave it as I learn more.
	if err := migrate.Seed(context.Background(), db); err != nil {
		return nil, fmt.Errorf("seeding db: %w", err)
	}

	return db, nil
}
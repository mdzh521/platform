package app

import (
	"fmt"
	"log"
	"time"

	"backend-center/internal/config"
	"backend-center/internal/domains/clusters/k8s"
	"backend-center/internal/domains/delivery/blueprints"
	"backend-center/internal/domains/delivery/cloud"
	"backend-center/internal/domains/delivery/summary"
	"backend-center/internal/domains/graph"
	"backend-center/internal/domains/iam/auth"
	"backend-center/internal/domains/iam/identitysource"
	"backend-center/internal/domains/iam/rbac"
	"backend-center/internal/domains/iam/user"
	"backend-center/internal/domains/machines/machine"
	"backend-center/internal/domains/platform/systemsetting"
	"backend-center/internal/domains/projects/project"
	platformcache "backend-center/internal/infra/cache"
	"backend-center/internal/infra/db"
	"backend-center/internal/infra/queue"
	platformruntime "backend-center/internal/infra/runtime"
	httpplatform "backend-center/internal/infra/web"
	"backend-center/internal/seed"
)

func Run() error {
	cfg := config.Load()
	log.Printf("starting %s on :%s (run_mode=%s, frontend_mode=%s, cache=%s, queue=%s)",
		cfg.AppName,
		cfg.Port,
		cfg.RunMode,
		cfg.Frontend.Mode,
		cfg.Cache.Driver,
		cfg.Queue.Driver,
	)

	db, err := database.Open(cfg)
	if err != nil {
		return err
	}
	if err := database.Migrate(db); err != nil {
		return err
	}
	if err := seed.Run(db, cfg); err != nil {
		return err
	}
	cacheStore, err := platformcache.New(cfg.Cache)
	if err != nil {
		return err
	}
	queueDispatcher, err := queue.New(cfg.Queue)
	if err != nil {
		return err
	}
	infra := &platformruntime.Infra{
		Cache: cacheStore,
		Queue: queueDispatcher,
	}
	deliveryDeps := cloud.NewDependencies(db, queueDispatcher, cfg.JWTSecret)

	userService := user.NewService(db)
	rbacService := rbac.NewService(db)
	identityService := identitysource.NewService(db)
	systemSettingService := systemsetting.NewService(db)
	projectService := project.NewService(db)
	graphService := graph.NewService(db)
	cloudService := cloud.NewService(
		deliveryDeps,
		summary.NewService(db),
		blueprints.NewService(db),
	)
	k8sService := k8s.NewService(db, cacheStore, queueDispatcher)
	machineService := machine.NewService(db, cacheStore, cfg.JWTSecret)
	authService := auth.NewService(cfg, userService, rbacService)
	if err := cloudService.Resources.RepairClusterEnrollmentScope(); err != nil {
		return err
	}
	if err := cloudService.Resources.RepairClusterEnvironmentCodes(); err != nil {
		return err
	}
	if err := cloudService.Addons.ReconcileExistingExecutions(); err != nil {
		return err
	}

	router := httpplatform.NewRouter(httpplatform.Dependencies{
		Config:                cfg,
		Infra:                 infra,
		AuthHandler:           auth.NewHandler(authService),
		UserHandler:           user.NewHandler(userService),
		IdentitySourceHandler: identitysource.NewHandler(identityService),
		RBACHandler:           rbac.NewHandler(rbacService),
		SystemSettingHandler:  systemsetting.NewHandler(systemSettingService),
		ProjectHandler:        project.NewHandler(projectService),
		GraphHandler:          graph.NewHandler(graphService),
		CloudHandler:          cloud.NewHandler(cloudService),
		K8sHandler:            k8s.NewHandler(k8sService, cfg.JWTSecret, cfg.TrustedOrigins),
		MachineHandler:        machine.NewHandler(machineService, cfg.JWTSecret, cfg.TrustedOrigins),
	})

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := cloudService.Deployments.RecoverTimedOutJobs(); err != nil {
				log.Printf("cloud timed-out job recovery failed: %v", err)
			}
			if err := cloudService.Addons.RecoverTimedOutExecutions(); err != nil {
				log.Printf("cluster addon timed-out execution recovery failed: %v", err)
			}
		}
	}()

	return router.Run(fmt.Sprintf(":%s", cfg.Port))
}

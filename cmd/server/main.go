package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"

	"http-proxy-platform/internal/config"
	"http-proxy-platform/internal/control"
	"http-proxy-platform/internal/proxy"
)

func main() {
	cfg := config.LoadFromEnv()
	var auth proxy.Authenticator
	var usage proxy.UsageRecorder
	var adminServer *control.APIServer
	var store *control.Store
	var err error

	if cfg.ControlPlaneEnabled {
		store, err = control.NewStore(cfg.DBPath, cfg.DeviceWindow)
		if err != nil {
			log.Fatalf("init control store failed: %v", err)
		}
		defer store.Close()

		if err := store.EnsureBootstrapUser(cfg.BootstrapUser, cfg.BootstrapPass); err != nil {
			log.Fatalf("bootstrap user failed: %v", err)
		}
		if err := store.EnsureBootstrapAdmins(cfg.BootstrapUser, cfg.AdminToken, cfg.BootstrapReadOnly, cfg.ReadOnlyAdminToken); err != nil {
			log.Fatalf("bootstrap admins failed: %v", err)
		}

		controlAuth := control.NewAuthenticator(store)
		auth = controlAuth
		usage = controlAuth
		adminServer = control.NewAPIServer(cfg, store)
	} else {
		staticAuth := proxy.NewStaticAuthenticator(cfg.Users)
		auth = staticAuth
		usage = staticAuth
	}

	httpServer := proxy.NewHTTPProxyServer(cfg, auth, usage)
	socksServer := proxy.NewSOCKS5Server(cfg, auth, usage)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := httpServer.Start(ctx); err != nil {
			log.Printf("http proxy stopped with error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := socksServer.Start(ctx); err != nil {
			log.Printf("socks5 proxy stopped with error: %v", err)
		}
	}()

	if adminServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := adminServer.Start(ctx); err != nil {
				log.Printf("admin api stopped with error: %v", err)
			}
		}()
	}

	log.Printf("proxy started: http=%s socks5=%s control_plane=%v admin=%s", cfg.HTTPListenAddr(), cfg.SOCKS5ListenAddr(), cfg.ControlPlaneEnabled, cfg.AdminListenAddr())
	<-ctx.Done()
	log.Printf("shutdown signal received")
	wg.Wait()
}

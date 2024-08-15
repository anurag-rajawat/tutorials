package main

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/anurag-rajawat/tutorials/nimbus/adapter/nimbus-kubearmor/manager"
)

func main() {
	ctx := ctrl.SetupSignalHandler()
	logger := setupLogger(ctx)
	manager.Run(ctx)
	<-ctx.Done()
	logger.Info("Shutting down")
	logger.Info("Shutdown complete")
}

func setupLogger(ctx context.Context) logr.Logger {
	ctrl.SetLogger(zap.New())
	logger := ctrl.Log
	ctrl.LoggerInto(ctx, logger)
	return logger
}

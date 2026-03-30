package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// DaemonServer interface for servers that support graceful shutdown
type DaemonServer interface {
	Shutdown(ctx context.Context) error
}

// SignalHandler manages signal handling and graceful shutdown
type SignalHandler struct {
	server         DaemonServer
	shutdownTimeout time.Duration
	logger         Logger
	pidFile        string
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler(server DaemonServer, timeout time.Duration, logger Logger) *SignalHandler {
	return &SignalHandler{
		server:         server,
		shutdownTimeout: timeout,
		logger:         logger,
	}
}

// SetPIDFile sets the PID file path for cleanup on shutdown
func (s *SignalHandler) SetPIDFile(path string) {
	s.pidFile = path
}

// SetupSignalHandler sets up signal handling for graceful shutdown
func (s *SignalHandler) SetupSignalHandler() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)

	// Register for SIGTERM (systemd stop), SIGINT (Ctrl+C), SIGHUP (reload config)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	go func() {
		sig := <-sigChan
		s.logger.Info(fmt.Sprintf("Received signal: %v, initiating graceful shutdown...", sig))

		// Handle SIGHUP specially - could trigger config reload
		if sig == syscall.SIGHUP {
			s.logger.Info("SIGHUP received - treating as shutdown signal")
		}

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer shutdownCancel()

		// Attempt graceful shutdown
		s.logger.Info(fmt.Sprintf("Starting graceful shutdown (timeout: %v)...", s.shutdownTimeout))
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error(fmt.Sprintf("Error during shutdown: %v", err))
		} else {
			s.logger.Info("Graceful shutdown completed successfully")
		}

		// Clean up PID file
		if s.pidFile != "" {
			if err := RemovePIDFile(s.pidFile); err != nil {
				s.logger.Warn(fmt.Sprintf("Failed to remove PID file: %v", err))
			}
		}

		// Cancel the main context
		cancel()
	}()

	return ctx
}

// SetupSignalHandlerSimple is a simplified version without server interface
func SetupSignalHandlerSimple(logger Logger, pidFile string, onShutdown func() error) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	go func() {
		sig := <-sigChan
		logger.Info(fmt.Sprintf("Received signal: %v, shutting down...", sig))

		if onShutdown != nil {
			if err := onShutdown(); err != nil {
				logger.Error(fmt.Sprintf("Shutdown error: %v", err))
			}
		}

		if pidFile != "" {
			if err := RemovePIDFile(pidFile); err != nil {
				logger.Warn(fmt.Sprintf("Failed to remove PID file: %v", err))
			}
		}

		cancel()
	}()

	return ctx
}

// WaitForSignal blocks until a signal is received or context is cancelled
func WaitForSignal(ctx context.Context) {
	<-ctx.Done()
}

package events

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"log/slog"
	"os"

	"github.com/r3labs/sse/v2"
	"github.com/rstudio/connect-client/internal/logging"
)

func NewLoggerWithSSE(debug bool) logging.Logger {
	eventServer := sse.New()
	eventServer.CreateStream("messages")
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	return NewLogger(logLevel, eventServer)
}

func NewLogger(level slog.Leveler, sseServer *sse.Server) logging.Logger {
	stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	if sseServer != nil {
		sseHandler := NewSSEHandler(sseServer, &SSEHandlerOptions{Level: level})
		multiHandler := logging.NewMultiHandler(stderrHandler, sseHandler)
		return logging.FromStdLogger(slog.New(multiHandler))
	} else {
		return logging.FromStdLogger(slog.New(stderrHandler))
	}
}

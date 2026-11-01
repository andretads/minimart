// Package cmd holds the entry points of the single MiniMart binary.
// One binary, several roles - the deployment platform chooses which by the command
// argument (server, worker, migrate, gateway, or dlq-replay). A plain switch keeps
// the course free of an additional command-framework dependency.
package cmd

import (
	"fmt"
	"log/slog"
	"minimart/conf"
	"minimart/pkg/logger"
	"os"
)

func Execute() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: minimart <server|worker|migrate|gateway|dlq-replay>")
		os.Exit(2)
	}

	logger.Setup(logger.Config{
		Level:   logger.ParseLevel(conf.LogLevel()),
		Format:  conf.LogFormat(),
		Service: "minimart",
		Version: version,
		Env:     conf.Env(),
	})

	var err error
	switch os.Args[1] {
	case "server":
		err = StartServer()
	case "worker":
		err = StartWorker()
	case "migrate":
		err = RunMigrate()
	case "gateway":
		err = StartGateway()
	case "dlq-replay":
		err = RunDLQReplay(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q; want server|worker|migrate|gateway|dlq-replay\n", os.Args[1])
		os.Exit(2)
	}

	if err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

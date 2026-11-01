package module

import (
	"context"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"
)

// Module is one self-contained domain in the modular monolith.
//
// The boot sequence in cmd/ calls these methods in phases, for every module:
//
//  1. Init             build the module's own internals (service, adapters).
//  2. RegisterContracts publish this module's contract into the shared registry.
//  3. (Verify)         the boot fails if any contract slot is still nil.
//     4a. RegisterHTTP    (server) wire this module's routes onto the mux.
//     4b. RegisterWorkers (worker) wire this module's Kafka consumers onto the worker.
//
// The server process runs phase 4a; the worker process runs phase 4b. No module
// may call another module before phase 3 completes.
type Module interface {
	Name() string
	Init(ctx context.Context) error
	RegisterContracts(ctx context.Context, c *contracts.Contracts) error
	RegisterHTTP(ctx context.Context, mux *http.ServeMux) error
	RegisterWorkers(ctx context.Context, w *worker.Worker) error
}

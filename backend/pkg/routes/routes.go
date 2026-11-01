// Package routes defines the contract the domain modules use to publish their
// HTTP routes.
//
// The modules register their routes on the shared mux through this interface
// rather than on *http.ServeMux directly, so the registered patterns can be
// observed from the outside (for example by the policy-coverage test that
// proves every route is governed by an authorization policy).
//
// *http.ServeMux satisfies Registrar, so the server keeps wiring routes onto a
// plain mux.
package routes

import "net/http"

// Registrar publishes HTTP routes. It is the subset of http.ServeMux the
// modules use.
type Registrar interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

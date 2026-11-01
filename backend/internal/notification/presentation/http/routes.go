package http

import (
	"minimart/pkg/routes"
)

// Register wires the notification routes onto the shared mux. Method-aware
// patterns (Go 1.22+) let the standard router dispatch without a framework.
// Notifications are written by the worker in reaction to events; HTTP clients
// can only read them. Preferences are the exception: users read and replace
// their own switches here.
func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("GET /v1/notifications", h.ListNotifications)
	mux.HandleFunc("GET /v1/notifications/preferences", h.GetPreferences)
	mux.HandleFunc("PUT /v1/notifications/preferences", h.UpdatePreferences)
	mux.HandleFunc("GET /v1/notifications/{id}", h.GetNotification)
	mux.HandleFunc("POST /v1/notifications/{id}/read", h.MarkNotificationRead)
	mux.HandleFunc("POST /v1/notifications/{id}/unread", h.MarkNotificationUnread)
	mux.HandleFunc("POST /v1/notifications/read-all", h.MarkAllNotificationsRead)
	mux.HandleFunc("DELETE /v1/notifications/{id}", h.DeleteNotification)
}

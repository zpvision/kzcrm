package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) apiRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/auth/me", s.me)
	r.Post("/auth/logout", s.logout)
	r.Get("/clients", s.listClients)
	r.Post("/clients", s.createClient)
	r.Get("/clients/{id}", s.getClient)
	r.Put("/clients/{id}", s.updateClient)
	r.Post("/clients/{id}/archive", s.archiveClient)
	r.Get("/clients/{id}/timeline", s.clientTimeline)
	r.Get("/clients/{id}/audit", s.clientAudit)
	r.Post("/payments", s.createPayment)
	r.Post("/payments/{id}/cancel", s.cancelPayment)
	r.Post("/renewals", s.createRenewal)
	r.Post("/renewals/{id}/cancel", s.cancelRenewal)
	r.Get("/counts", s.listCounts)
	r.Put("/counts/{clientID}", s.saveCount)
	r.Post("/counts/copy", s.copyCounts)
	r.Post("/counts/confirm", s.confirmCounts)
	return r
}

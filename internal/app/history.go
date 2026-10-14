package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
)

func (s *Server) clientTimeline(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "Некорректный ID")
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT kind,id,service,event_date,period,months_count,amount,currency,state,user_name,comment,payment_id FROM (
	 SELECT 'payment' kind,p.id::text,p.service,p.created_at event_date,to_char(p.billing_month,'YYYY-MM')||' · '||p.months_count||' мес.' period,p.months_count,p.amount,p.currency::text,p.state::text,u.display_name user_name,p.comment,NULL::text payment_id FROM payments p JOIN users u ON u.id=p.created_by WHERE p.client_id=$1
	 UNION ALL SELECT 'renewal',r.id::text,r.service,r.created_at,COALESCE(to_char(r.previous_expires_at,'DD.MM.YYYY'),'—')||' → '||to_char(r.new_expires_at,'DD.MM.YYYY'),r.months_count,NULL,NULL,r.state::text,u.display_name,r.comment,r.payment_id::text FROM renewals r JOIN users u ON u.id=r.created_by WHERE r.client_id=$1) x ORDER BY event_date DESC`, id)
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var kind, rid, service, period, state, user string
		var date any
		var months int
		var amount, currency, comment, paymentID any
		if err = rows.Scan(&kind, &rid, &service, &date, &period, &months, &amount, &currency, &state, &user, &comment, &paymentID); err != nil {
			dbProblem(w, err)
			return
		}
		out = append(out, map[string]any{"kind": kind, "id": rid, "service": service, "date": date, "period": period, "months": months, "amount": amount, "currency": currency, "state": state, "user": user, "comment": comment, "paymentId": paymentID})
	}
	respond(w, 200, out)
}
func (s *Server) clientAudit(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "Некорректный ID")
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT a.id,a.action,a.entity_type,a.old_values,a.new_values,a.created_at,u.display_name FROM audit_logs a JOIN users u ON u.id=a.user_id WHERE a.client_id=$1 ORDER BY a.created_at DESC LIMIT 250`, id)
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var aid int64
		var action, entity, user string
		var old, newVal, date any
		if err = rows.Scan(&aid, &action, &entity, &old, &newVal, &date, &user); err != nil {
			dbProblem(w, err)
			return
		}
		out = append(out, map[string]any{"id": aid, "action": action, "entity": entity, "old": old, "new": newVal, "date": date, "user": user})
	}
	respond(w, 200, out)
}

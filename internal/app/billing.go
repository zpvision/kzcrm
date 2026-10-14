package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type paymentInput struct {
	ClientID       uuid.UUID `json:"clientId"`
	Service        string    `json:"service"`
	ReceivedOn     string    `json:"receivedOn"`
	BillingMonth   string    `json:"billingMonth"`
	Months         int       `json:"months"`
	Amount         *float64  `json:"amount"`
	Comment        string    `json:"comment"`
	IdempotencyKey uuid.UUID `json:"idempotencyKey"`
}

func serviceOK(v string) bool { return v == "finkoper" || v == "whatsapp" }
func (s *Server) createPayment(w http.ResponseWriter, r *http.Request) {
	var in paymentInput
	if !decode(w, r, &in) {
		return
	}
	if !serviceOK(in.Service) || in.Months < 1 || in.Months > 120 || in.IdempotencyKey == uuid.Nil {
		problem(w, 422, "Проверьте услугу, срок и ключ операции")
		return
	}
	received, e1 := time.Parse("2006-01-02", in.ReceivedOn)
	month, e2 := validMonth(in.BillingMonth)
	if e1 != nil || e2 != nil {
		problem(w, 422, "Проверьте даты")
		return
	}
	u := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var duplicate uuid.UUID
	err = tx.QueryRow(r.Context(), `SELECT id FROM payments WHERE client_id=$1 AND service=$2 AND received_on=$3 AND billing_month=$4 AND months_count=$5 AND state='active' LIMIT 1`, in.ClientID, in.Service, received, month, in.Months).Scan(&duplicate)
	if err == nil {
		problem(w, 409, "Похожая оплата уже зарегистрирована")
		return
	}
	if err != pgx.ErrNoRows {
		dbProblem(w, err)
		return
	}
	var id uuid.UUID
	err = tx.QueryRow(r.Context(), `INSERT INTO payments(client_id,service,received_on,billing_month,months_count,amount,comment,created_by,idempotency_key) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9) RETURNING id`, in.ClientID, in.Service, received, month, in.Months, in.Amount, in.Comment, u.ID, in.IdempotencyKey).Scan(&id)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO payment_allocations(payment_id,billing_month) SELECT $1,(date_trunc('month',$2::date)+make_interval(months=>n))::date FROM generate_series(0,$3-1) n`, id, month, in.Months)
	}
	if err == nil {
		b, _ := json.Marshal(in)
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_logs(client_id,user_id,action,entity_type,entity_id,new_values) VALUES($1,$2,'payment.created','payment',$3,$4)`, in.ClientID, u.ID, id.String(), b)
	}
	if err != nil {
		dbProblem(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		dbProblem(w, err)
		return
	}
	respond(w, 201, map[string]any{"id": id})
}

type reasonInput struct {
	Reason string `json:"reason"`
}

func (s *Server) cancelPayment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "Некорректный ID")
		return
	}
	var in reasonInput
	if !decode(w, r, &in) {
		return
	}
	if len(in.Reason) < 3 {
		problem(w, 422, "Укажите причину отмены")
		return
	}
	u := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var clientID uuid.UUID
	var linked bool
	err = tx.QueryRow(r.Context(), `SELECT p.client_id,EXISTS(SELECT 1 FROM renewals r WHERE r.payment_id=p.id AND r.state='active') FROM payments p WHERE p.id=$1 AND p.state='active' FOR UPDATE`, id).Scan(&clientID, &linked)
	if err != nil {
		dbProblem(w, err)
		return
	}
	if linked {
		problem(w, 409, "Сначала отмените связанное продление")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE payments SET state='cancelled',cancellation_reason=$2,cancelled_at=now(),cancelled_by=$3 WHERE id=$1`, id, in.Reason, u.ID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_logs(client_id,user_id,action,entity_type,entity_id,new_values) VALUES($1,$2,'payment.cancelled','payment',$3,jsonb_build_object('reason',$4::text))`, clientID, u.ID, id.String(), in.Reason)
	}
	if err != nil {
		dbProblem(w, err)
		return
	}
	_ = tx.Commit(r.Context())
	respond(w, 200, map[string]any{"ok": true})
}

type renewalInput struct {
	ClientID       uuid.UUID  `json:"clientId"`
	Service        string     `json:"service"`
	Months         int        `json:"months"`
	NewExpires     *string    `json:"newExpires"`
	PaymentID      *uuid.UUID `json:"paymentId"`
	Comment        string     `json:"comment"`
	IdempotencyKey uuid.UUID  `json:"idempotencyKey"`
}

func AddCalendarMonths(d time.Time, months int) time.Time {
	target := time.Date(d.Year(), d.Month()+time.Month(months), 1, 0, 0, 0, 0, d.Location())
	last := target.AddDate(0, 1, -1).Day()
	day := d.Day()
	if day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, d.Location())
}
func (s *Server) createRenewal(w http.ResponseWriter, r *http.Request) {
	var in renewalInput
	if !decode(w, r, &in) {
		return
	}
	if !serviceOK(in.Service) || in.Months < 1 || in.Months > 120 || in.IdempotencyKey == uuid.Nil {
		problem(w, 422, "Проверьте услугу, срок и ключ операции")
		return
	}
	u := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var subID uuid.UUID
	var old *time.Time
	var enabled bool
	err = tx.QueryRow(r.Context(), `SELECT id,expires_at,enabled FROM subscriptions WHERE client_id=$1 AND service=$2 FOR UPDATE`, in.ClientID, in.Service).Scan(&subID, &old, &enabled)
	if err != nil {
		dbProblem(w, err)
		return
	}
	if !enabled {
		problem(w, 422, "Интеграция WhatsApp выключена")
		return
	}
	base := time.Now()
	if old != nil {
		base = *old
	}
	next := AddCalendarMonths(base, in.Months)
	if in.NewExpires != nil && *in.NewExpires != "" {
		next, err = time.Parse("2006-01-02", *in.NewExpires)
		if err != nil {
			problem(w, 422, "Некорректная новая дата")
			return
		}
		if !next.After(base) {
			problem(w, 422, "Новая дата должна быть позже текущей")
			return
		}
	}
	if in.PaymentID != nil {
		var ok bool
		err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM payments WHERE id=$1 AND client_id=$2 AND service=$3 AND months_count=$4 AND state='active' AND NOT EXISTS(SELECT 1 FROM renewals WHERE payment_id=$1 AND state='active'))`, *in.PaymentID, in.ClientID, in.Service, in.Months).Scan(&ok)
		if err != nil || !ok {
			problem(w, 422, "Оплата не найдена, уже использована или её срок не совпадает с продлением")
			return
		}
	}
	var id uuid.UUID
	err = tx.QueryRow(r.Context(), `INSERT INTO renewals(client_id,subscription_id,service,previous_expires_at,new_expires_at,months_count,payment_id,comment,created_by,idempotency_key) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,$10) RETURNING id`, in.ClientID, subID, in.Service, old, next, in.Months, in.PaymentID, in.Comment, u.ID, in.IdempotencyKey).Scan(&id)
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE subscriptions SET expires_at=$2,version=version+1,updated_at=now() WHERE id=$1`, subID, next)
	}
	if err == nil {
		b, _ := json.Marshal(in)
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_logs(client_id,user_id,action,entity_type,entity_id,new_values) VALUES($1,$2,'renewal.created','renewal',$3,$4)`, in.ClientID, u.ID, id.String(), b)
	}
	if err != nil {
		dbProblem(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		dbProblem(w, err)
		return
	}
	respond(w, 201, map[string]any{"id": id, "newExpires": next.Format("2006-01-02")})
}

func (s *Server) cancelRenewal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "Некорректный ID")
		return
	}
	var in reasonInput
	if !decode(w, r, &in) {
		return
	}
	if len(in.Reason) < 3 {
		problem(w, 422, "Укажите причину отмены")
		return
	}
	u := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var clientID, subID uuid.UUID
	var previous *time.Time
	var current, newDate time.Time
	err = tx.QueryRow(r.Context(), `SELECT r.client_id,r.subscription_id,r.previous_expires_at,r.new_expires_at,s.expires_at FROM renewals r JOIN subscriptions s ON s.id=r.subscription_id WHERE r.id=$1 AND r.state='active' FOR UPDATE OF r,s`, id).Scan(&clientID, &subID, &previous, &newDate, &current)
	if err != nil {
		dbProblem(w, err)
		return
	}
	if !current.Equal(newDate) {
		problem(w, 409, "После этого продления были другие изменения. Отмените их сначала")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE renewals SET state='cancelled',cancellation_reason=$2,cancelled_at=now(),cancelled_by=$3 WHERE id=$1`, id, in.Reason, u.ID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE subscriptions SET expires_at=$2,version=version+1,updated_at=now() WHERE id=$1`, subID, previous)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_logs(client_id,user_id,action,entity_type,entity_id,new_values) VALUES($1,$2,'renewal.cancelled','renewal',$3,jsonb_build_object('reason',$4::text))`, clientID, u.ID, id.String(), in.Reason)
	}
	if err != nil {
		dbProblem(w, err)
		return
	}
	_ = tx.Commit(r.Context())
	respond(w, 200, map[string]any{"ok": true})
}

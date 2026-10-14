package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type clientInput struct {
	CompanyName       string  `json:"companyName"`
	TaxID             string  `json:"taxId"`
	Email             string  `json:"email"`
	ContactName       string  `json:"contactName"`
	Phone             string  `json:"phone"`
	Comment           string  `json:"comment"`
	FinKoperExpires   *string `json:"finKoperExpires"`
	WhatsAppEnabled   bool    `json:"whatsAppEnabled"`
	WhatsAppEnabledAt *string `json:"whatsAppEnabledAt"`
	WhatsAppExpires   *string `json:"whatsAppExpires"`
	Version           int     `json:"version"`
}
type clientRow struct {
	ID          uuid.UUID  `json:"id"`
	CompanyName string     `json:"companyName"`
	TaxID       *string    `json:"taxId"`
	Email       string     `json:"email"`
	ContactName *string    `json:"contactName"`
	Phone       *string    `json:"phone"`
	Comment     *string    `json:"comment"`
	Count       *int       `json:"count"`
	FinExpires  *time.Time `json:"finKoperExpires"`
	WAEnabled   bool       `json:"whatsAppEnabled"`
	WAEnabledAt *time.Time `json:"whatsAppEnabledAt"`
	WAExpires   *time.Time `json:"whatsAppExpires"`
	FinStatus   string     `json:"finKoperStatus"`
	WAStatus    string     `json:"whatsAppStatus"`
	Version     int        `json:"version"`
	ArchivedAt  *time.Time `json:"archivedAt"`
}

const clientSelect = `SELECT c.id,c.company_name,c.tax_id,c.email,c.contact_name,c.phone,c.comment,mc.count,
 sf.expires_at,COALESCE(sw.enabled,false),sw.enabled_at,sw.expires_at,
 CASE WHEN pf.id IS NULL THEN 'unpaid' WHEN rf.id IS NULL THEN 'pending' ELSE 'completed' END,
 CASE WHEN pw.id IS NULL THEN 'unpaid' WHEN rw.id IS NULL THEN 'pending' ELSE 'completed' END,c.version,c.archived_at
 FROM clients c
 LEFT JOIN monthly_client_counts mc ON mc.client_id=c.id AND mc.month=$1
 LEFT JOIN subscriptions sf ON sf.client_id=c.id AND sf.service='finkoper'
 LEFT JOIN subscriptions sw ON sw.client_id=c.id AND sw.service='whatsapp'
 LEFT JOIN LATERAL (SELECT p.id FROM payments p JOIN payment_allocations pa ON pa.payment_id=p.id WHERE p.client_id=c.id AND p.service='finkoper' AND p.state='active' AND pa.billing_month=$1 ORDER BY p.created_at DESC LIMIT 1) pf ON true
 LEFT JOIN LATERAL (SELECT id FROM renewals WHERE payment_id=pf.id AND state='active' LIMIT 1) rf ON true
 LEFT JOIN LATERAL (SELECT p.id FROM payments p JOIN payment_allocations pa ON pa.payment_id=p.id WHERE p.client_id=c.id AND p.service='whatsapp' AND p.state='active' AND pa.billing_month=$1 ORDER BY p.created_at DESC LIMIT 1) pw ON true
 LEFT JOIN LATERAL (SELECT id FROM renewals WHERE payment_id=pw.id AND state='active' LIMIT 1) rw ON true`

func validMonth(v string) (time.Time, error) { return time.Parse("2006-01-02", v+"-01") }
func scanClient(row pgx.Row) (clientRow, error) {
	var c clientRow
	err := row.Scan(&c.ID, &c.CompanyName, &c.TaxID, &c.Email, &c.ContactName, &c.Phone, &c.Comment, &c.Count, &c.FinExpires, &c.WAEnabled, &c.WAEnabledAt, &c.WAExpires, &c.FinStatus, &c.WAStatus, &c.Version, &c.ArchivedAt)
	return c, err
}
func (s *Server) listClients(w http.ResponseWriter, r *http.Request) {
	m, err := validMonth(r.URL.Query().Get("month"))
	if err != nil {
		problem(w, 400, "Месяц должен иметь формат YYYY-MM")
		return
	}
	q := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	rows, err := s.db.Query(r.Context(), clientSelect+` WHERE c.archived_at IS NULL AND (c.company_name ILIKE $2 OR COALESCE(c.tax_id,'') ILIKE $2 OR c.email ILIKE $2 OR COALESCE(c.phone,'') ILIKE $2) ORDER BY c.company_name`, m, q)
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer rows.Close()
	out := []clientRow{}
	for rows.Next() {
		c, e := scanClient(rows)
		if e != nil {
			dbProblem(w, e)
			return
		}
		out = append(out, c)
	}
	respond(w, 200, out)
}

func validateClient(in clientInput) string {
	if strings.TrimSpace(in.CompanyName) == "" {
		return "Укажите название компании"
	}
	if strings.TrimSpace(in.Email) == "" {
		return "Укажите почту / логин"
	}
	if in.WhatsAppEnabled && in.WhatsAppEnabledAt == nil {
		return "Укажите дату включения WhatsApp"
	}
	return ""
}
func dateValue(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}
func (s *Server) createClient(w http.ResponseWriter, r *http.Request) {
	var in clientInput
	if !decode(w, r, &in) {
		return
	}
	if msg := validateClient(in); msg != "" {
		problem(w, 422, msg)
		return
	}
	u := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var id uuid.UUID
	err = tx.QueryRow(r.Context(), `INSERT INTO clients(company_name,tax_id,email,contact_name,phone,comment,created_by,updated_by) VALUES($1,NULLIF($2,''),lower($3),NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),$7,$7) RETURNING id`, strings.TrimSpace(in.CompanyName), in.TaxID, strings.TrimSpace(in.Email), in.ContactName, in.Phone, in.Comment, u.ID).Scan(&id)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO subscriptions(client_id,service,enabled,enabled_at,expires_at) VALUES($1,'finkoper',true,NULL,$2),($1,'whatsapp',$3,$4,$5)`, id, dateValue(in.FinKoperExpires), in.WhatsAppEnabled, dateValue(in.WhatsAppEnabledAt), dateValue(in.WhatsAppExpires))
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_logs(client_id,user_id,action,entity_type,entity_id,new_values) VALUES($1,$2,'client.created','client',$1,$3)`, id, u.ID, in)
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

func (s *Server) getClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "Некорректный ID")
		return
	}
	m := time.Now()
	c, err := scanClient(s.db.QueryRow(r.Context(), clientSelect+` WHERE c.id=$2`, time.Date(m.Year(), m.Month(), 1, 0, 0, 0, 0, time.UTC), id))
	if err != nil {
		dbProblem(w, err)
		return
	}
	respond(w, 200, c)
}
func (s *Server) updateClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "Некорректный ID")
		return
	}
	var in clientInput
	if !decode(w, r, &in) {
		return
	}
	if msg := validateClient(in); msg != "" {
		problem(w, 422, msg)
		return
	}
	u := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var old json.RawMessage
	err = tx.QueryRow(r.Context(), `SELECT to_jsonb(c) FROM clients c WHERE id=$1 FOR UPDATE`, id).Scan(&old)
	if err == nil {
		tag, e := tx.Exec(r.Context(), `UPDATE clients SET company_name=$2,tax_id=NULLIF($3,''),email=lower($4),contact_name=NULLIF($5,''),phone=NULLIF($6,''),comment=NULLIF($7,''),updated_by=$8,updated_at=now(),version=version+1 WHERE id=$1 AND version=$9`, id, in.CompanyName, in.TaxID, in.Email, in.ContactName, in.Phone, in.Comment, u.ID, in.Version)
		err = e
		if err == nil && tag.RowsAffected() == 0 {
			problem(w, 409, "Данные уже изменены другим пользователем. Обновите страницу")
			return
		}
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE subscriptions SET expires_at=$3,enabled=CASE WHEN service='whatsapp' THEN $4 ELSE true END,enabled_at=CASE WHEN service='whatsapp' THEN $5 ELSE enabled_at END,version=version+1,updated_at=now() WHERE client_id=$1 AND service=$2`, id, "finkoper", dateValue(in.FinKoperExpires), in.WhatsAppEnabled, dateValue(in.WhatsAppEnabledAt))
		if err == nil {
			_, err = tx.Exec(r.Context(), `UPDATE subscriptions SET enabled=$2,enabled_at=$3,expires_at=$4,version=version+1,updated_at=now() WHERE client_id=$1 AND service='whatsapp'`, id, in.WhatsAppEnabled, dateValue(in.WhatsAppEnabledAt), dateValue(in.WhatsAppExpires))
		}
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_logs(client_id,user_id,action,entity_type,entity_id,old_values,new_values) VALUES($1,$2,'client.updated','client',$1,$3,$4)`, id, u.ID, old, in)
	}
	if err != nil {
		dbProblem(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		dbProblem(w, err)
		return
	}
	respond(w, 200, map[string]any{"ok": true})
}
func (s *Server) archiveClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "Некорректный ID")
		return
	}
	u := currentUser(r)
	tag, err := s.db.Exec(r.Context(), `WITH changed AS (UPDATE clients SET archived_at=now(),updated_by=$2,version=version+1 WHERE id=$1 AND archived_at IS NULL RETURNING id) INSERT INTO audit_logs(client_id,user_id,action,entity_type,entity_id,new_values) SELECT id,$2,'client.archived','client',id,'{"archived":true}' FROM changed`, id, u.ID)
	if err != nil {
		dbProblem(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		problem(w, 409, "Клиент уже архивирован")
		return
	}
	respond(w, 200, map[string]any{"ok": true})
}

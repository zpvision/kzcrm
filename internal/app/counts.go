package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type countRow struct {
	ClientID    uuid.UUID `json:"clientId"`
	CompanyName string    `json:"companyName"`
	Email       string    `json:"email"`
	TaxID       *string   `json:"taxId"`
	Previous    *int      `json:"previous"`
	Count       *int      `json:"count"`
	Version     *int      `json:"version"`
}

func (s *Server) listCounts(w http.ResponseWriter, r *http.Request) {
	month, err := validMonth(r.URL.Query().Get("month"))
	if err != nil {
		problem(w, 400, "Месяц должен иметь формат YYYY-MM")
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT c.id,c.company_name,c.email,c.tax_id,p.count,m.count,m.version FROM clients c LEFT JOIN monthly_client_counts m ON m.client_id=c.id AND m.month=$1 LEFT JOIN monthly_client_counts p ON p.client_id=c.id AND p.month=($1::date-interval '1 month')::date WHERE c.archived_at IS NULL ORDER BY c.company_name`, month)
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer rows.Close()
	items := []countRow{}
	for rows.Next() {
		var x countRow
		if err = rows.Scan(&x.ClientID, &x.CompanyName, &x.Email, &x.TaxID, &x.Previous, &x.Count, &x.Version); err != nil {
			dbProblem(w, err)
			return
		}
		items = append(items, x)
	}
	var confirmedAt *time.Time
	var by *string
	_ = s.db.QueryRow(r.Context(), `SELECT mc.confirmed_at,u.display_name FROM monthly_confirmations mc JOIN users u ON u.id=mc.confirmed_by WHERE month=$1`, month).Scan(&confirmedAt, &by)
	total := 0
	for _, x := range items {
		if x.Count != nil {
			total += *x.Count
		}
	}
	respond(w, 200, map[string]any{"items": items, "total": total, "confirmedAt": confirmedAt, "confirmedBy": by})
}

type countInput struct {
	Count             *int `json:"count"`
	Version           *int `json:"version"`
	ConfirmCorrection bool `json:"confirmCorrection"`
}

func (s *Server) saveCount(w http.ResponseWriter, r *http.Request) {
	clientID, err := uuid.Parse(chi.URLParam(r, "clientID"))
	if err != nil {
		problem(w, 400, "Некорректный ID")
		return
	}
	month, err := validMonth(r.URL.Query().Get("month"))
	if err != nil {
		problem(w, 400, "Некорректный месяц")
		return
	}
	var in countInput
	if !decode(w, r, &in) {
		return
	}
	if in.Count != nil && *in.Count < 0 {
		problem(w, 422, "Количество не может быть отрицательным")
		return
	}
	u := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var confirmed bool
	_ = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM monthly_confirmations WHERE month=$1)`, month).Scan(&confirmed)
	if confirmed && !in.ConfirmCorrection {
		problem(w, 409, "Месяц подтверждён. Подтвердите корректировку")
		return
	}
	var old *int
	var version int
	err = tx.QueryRow(r.Context(), `SELECT count,version FROM monthly_client_counts WHERE client_id=$1 AND month=$2 FOR UPDATE`, clientID, month).Scan(&old, &version)
	if err == pgx.ErrNoRows {
		_, err = tx.Exec(r.Context(), `INSERT INTO monthly_client_counts(client_id,month,count,updated_by) VALUES($1,$2,$3,$4)`, clientID, month, in.Count, u.ID)
	} else if err == nil {
		if in.Version == nil || *in.Version != version {
			problem(w, 409, "Значение уже изменено другим пользователем")
			return
		}
		_, err = tx.Exec(r.Context(), `UPDATE monthly_client_counts SET count=$3,version=version+1,updated_by=$4,updated_at=now() WHERE client_id=$1 AND month=$2`, clientID, month, in.Count, u.ID)
	} else {
		dbProblem(w, err)
		return
	}
	if err == nil {
		action := "count.updated"
		if confirmed {
			action = "count.corrected"
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_logs(client_id,user_id,action,entity_type,entity_id,old_values,new_values) VALUES($1,$2,$3,'monthly_client_count',$4,jsonb_build_object('count',$5::integer),jsonb_build_object('count',$6::integer,'month',$7::date))`, clientID, u.ID, action, clientID.String()+":"+month.Format("2006-01"), old, in.Count, month)
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

type monthAction struct {
	Month     string `json:"month"`
	Overwrite bool   `json:"overwrite"`
}

func (s *Server) copyCounts(w http.ResponseWriter, r *http.Request) {
	var in monthAction
	if !decode(w, r, &in) {
		return
	}
	month, err := validMonth(in.Month)
	if err != nil {
		problem(w, 422, "Некорректный месяц")
		return
	}
	u := currentUser(r)
	condition := "AND dst.id IS NULL"
	if in.Overwrite {
		condition = ""
	}
	query := `INSERT INTO monthly_client_counts(client_id,month,count,updated_by) SELECT c.id,$1,src.count,$2 FROM clients c JOIN monthly_client_counts src ON src.client_id=c.id AND src.month=($1::date-interval '1 month')::date LEFT JOIN monthly_client_counts dst ON dst.client_id=c.id AND dst.month=$1 WHERE c.archived_at IS NULL ` + condition + ` ON CONFLICT(client_id,month) DO UPDATE SET count=excluded.count,updated_by=excluded.updated_by,updated_at=now(),version=monthly_client_counts.version+1`
	tag, err := s.db.Exec(r.Context(), query, month, u.ID)
	if err != nil {
		dbProblem(w, err)
		return
	}
	respond(w, 200, map[string]any{"copied": tag.RowsAffected()})
}
func (s *Server) confirmCounts(w http.ResponseWriter, r *http.Request) {
	var in monthAction
	if !decode(w, r, &in) {
		return
	}
	month, err := validMonth(in.Month)
	if err != nil {
		problem(w, 422, "Некорректный месяц")
		return
	}
	u := currentUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		dbProblem(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var missing int
	err = tx.QueryRow(r.Context(), `SELECT count(*) FROM clients c LEFT JOIN monthly_client_counts m ON m.client_id=c.id AND m.month=$1 WHERE c.archived_at IS NULL AND m.count IS NULL`, month).Scan(&missing)
	if err != nil {
		dbProblem(w, err)
		return
	}
	if missing > 0 {
		problem(w, 422, "Заполните количество для всех клиентов")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO monthly_confirmations(month,confirmed_by) VALUES($1,$2) ON CONFLICT(month) DO UPDATE SET confirmed_by=excluded.confirmed_by,confirmed_at=now(),revision=monthly_confirmations.revision+1`, month, u.ID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_logs(user_id,action,entity_type,entity_id,new_values) VALUES($1,'month.confirmed','monthly_confirmation',$2,jsonb_build_object('month',$3::text))`, u.ID, in.Month, in.Month)
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

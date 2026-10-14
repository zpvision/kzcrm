package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		problem(w, 400, "Некорректные данные: "+err.Error())
		return false
	}
	return true
}
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func problem(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]any{"error": message})
}
func dbProblem(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		problem(w, 404, "Запись не найдена")
	} else if strings.Contains(err.Error(), "duplicate key") {
		problem(w, 409, "Такая запись уже существует")
	} else {
		problem(w, 500, "Ошибка базы данных")
	}
}

package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitUsers(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" || cfg.PartnerEmail == "" || cfg.PartnerPassword == "" {
		return fmt.Errorf("ADMIN_EMAIL, ADMIN_PASSWORD, PARTNER_EMAIL and PARTNER_PASSWORD are required")
	}
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	users := []struct{ email, password, name, role string }{{cfg.AdminEmail, cfg.AdminPassword, "Администратор FinKoper", "admin"}, {cfg.PartnerEmail, cfg.PartnerPassword, "Партнёр Казахстан", "partner"}}
	for _, u := range users {
		if len(u.password) < 12 {
			return fmt.Errorf("password for %s must have at least 12 characters", u.email)
		}
		h, e := hashPassword(u.password)
		if e != nil {
			return e
		}
		_, e = db.Exec(ctx, `INSERT INTO users(email,password_hash,display_name,role) VALUES($1,$2,$3,$4) ON CONFLICT(email) DO UPDATE SET password_hash=excluded.password_hash,display_name=excluded.display_name,role=excluded.role,active=true,updated_at=now()`, strings.ToLower(strings.TrimSpace(u.email)), h, u.name, u.role)
		if e != nil {
			return e
		}
	}
	return nil
}

package app

import (
	"strings"
	"testing"
	"time"
)

func TestAddCalendarMonthsClampsEndOfMonth(t *testing.T) {
	tests := []struct {
		from   string
		months int
		want   string
	}{{"2026-01-31", 1, "2026-02-28"}, {"2028-01-31", 1, "2028-02-29"}, {"2026-10-30", 1, "2026-11-30"}, {"2026-08-31", 6, "2027-02-28"}, {"2026-02-28", 12, "2027-02-28"}}
	for _, tt := range tests {
		d, _ := time.Parse("2006-01-02", tt.from)
		got := AddCalendarMonths(d, tt.months).Format("2006-01-02")
		if got != tt.want {
			t.Errorf("AddCalendarMonths(%s,%d)=%s want %s", tt.from, tt.months, got, tt.want)
		}
	}
}
func TestPasswordArgon2id(t *testing.T) {
	h, err := hashPassword("long secure password")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Fatalf("unexpected hash: %s", h)
	}
	if !verifyPassword("long secure password", h) {
		t.Fatal("correct password rejected")
	}
	if verifyPassword("wrong", h) {
		t.Fatal("wrong password accepted")
	}
}
func TestServicesAreIndependent(t *testing.T) {
	if !serviceOK("finkoper") || !serviceOK("whatsapp") {
		t.Fatal("supported service rejected")
	}
	if serviceOK("other") {
		t.Fatal("unknown service accepted")
	}
}
func TestAnnualPeriodHasTwelveDistinctMonths(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-10-01")
	seen := map[string]bool{}
	for i := 0; i < 12; i++ {
		seen[AddCalendarMonths(start, i).Format("2006-01")] = true
	}
	if len(seen) != 12 {
		t.Fatalf("got %d months", len(seen))
	}
}

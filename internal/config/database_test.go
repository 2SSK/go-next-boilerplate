package config

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestDatabaseURL(t *testing.T) {
	t.Run("built from libpq variables, special characters intact", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		t.Setenv("PGHOST", "db.internal")
		t.Setenv("PGPORT", "6543")
		t.Setenv("PGUSER", "svc user")
		t.Setenv("PGPASSWORD", `p@ss $word #1 'x' "y" /?&=:%`)
		t.Setenv("PGDATABASE", "shop")
		t.Setenv("PGSSLMODE", "require")

		parsed, err := pgx.ParseConfig(databaseURL())
		if err != nil {
			t.Fatalf("ParseConfig: %v", err)
		}
		if parsed.Password != `p@ss $word #1 'x' "y" /?&=:%` {
			t.Errorf("password = %q", parsed.Password)
		}
		if parsed.User != "svc user" || parsed.Host != "db.internal" || parsed.Port != 6543 || parsed.Database != "shop" {
			t.Errorf("parsed = user %q host %q port %d db %q", parsed.User, parsed.Host, parsed.Port, parsed.Database)
		}
	})

	t.Run("defaults match docker-compose.yml", func(t *testing.T) {
		for _, k := range []string{"DATABASE_URL", "PGHOST", "PGPORT", "PGUSER", "PGPASSWORD", "PGDATABASE", "PGSSLMODE"} {
			t.Setenv(k, "")
		}
		//nolint:gosec // app:app is the documented local default from docker-compose.yml, not a secret
		if got, want := databaseURL(), "postgres://app:app@localhost:5432/app?sslmode=disable"; got != want {
			t.Errorf("databaseURL() = %q, want %q", got, want)
		}
	})

	t.Run("DATABASE_URL wins when set", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://hosted.example/db")
		t.Setenv("PGHOST", "ignored")
		if got := databaseURL(); got != "postgres://hosted.example/db" {
			t.Errorf("databaseURL() = %q", got)
		}
	})
}

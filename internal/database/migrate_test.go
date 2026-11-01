package database

import (
	"strings"
	"testing"
	"testing/fstest"
)

func files(t *testing.T, m fstest.MapFS) []Migration {
	t.Helper()
	got, err := Load(m)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return got
}

func TestLoad(t *testing.T) {
	got := files(t, fstest.MapFS{
		"0002_b.sql": {Data: []byte("SELECT 2;")},
		"0001_a.sql": {Data: []byte("SELECT 1;")},
		"README.md":  {Data: []byte("not a migration")},
	})
	if len(got) != 2 || got[0].Version != "0001_a.sql" || got[1].Version != "0002_b.sql" {
		t.Fatalf("Load = %+v, want 0001 then 0002 and no README", got)
	}
	if len(got[0].Checksum) != 64 || got[0].Checksum == got[1].Checksum {
		t.Errorf("checksums look wrong: %q %q", got[0].Checksum, got[1].Checksum)
	}
}

func TestPlan(t *testing.T) {
	fs := fstest.MapFS{
		"0001_a.sql": {Data: []byte("SELECT 1;")},
		"0002_b.sql": {Data: []byte("SELECT 2;")},
		"0003_c.sql": {Data: []byte("SELECT 3;")},
	}
	all := files(t, fs)
	rec := func(i int) Applied { return Applied{Version: all[i].Version, Checksum: all[i].Checksum} }

	tests := []struct {
		name        string
		files       []Migration
		applied     []Applied
		wantPending []string
		wantErr     string
	}{
		{"fresh database: everything pending", all, nil, []string{"0001_a.sql", "0002_b.sql", "0003_c.sql"}, ""},
		{"partly applied: the rest pending", all, []Applied{rec(0)}, []string{"0002_b.sql", "0003_c.sql"}, ""},
		{"fully applied: nothing pending", all, []Applied{rec(0), rec(1), rec(2)}, nil, ""},
		{
			"applied file edited",
			all, []Applied{{Version: "0001_a.sql", Checksum: "deadbeef"}}, nil,
			"0001_a.sql was modified after it was applied",
		},
		{
			"applied file deleted",
			all[1:], []Applied{rec(0)}, nil,
			"0001_a.sql was applied but its file is gone",
		},
		{
			"new file sorts before an applied one",
			all, []Applied{rec(0), rec(2)}, nil,
			"0002_b.sql is new but sorts before the applied 0003_c.sql",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := plan(tt.files, tt.applied)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("plan: %v", err)
			}
			var got []string
			for _, m := range p.Pending {
				got = append(got, m.Version)
			}
			if strings.Join(got, ",") != strings.Join(tt.wantPending, ",") {
				t.Errorf("pending = %v, want %v", got, tt.wantPending)
			}
		})
	}
}

func TestEmbeddedMigrationsLoad(t *testing.T) {
	got := mustLoad(t)
	if len(got) == 0 || got[0].Version != "0001_init.sql" {
		t.Fatalf("embedded migrations = %+v, want 0001_init.sql first", got)
	}
}

func mustLoad(t *testing.T) []Migration {
	t.Helper()
	got, err := Load(Migrations())
	if err != nil {
		t.Fatalf("Load(Migrations()): %v", err)
	}
	return got
}

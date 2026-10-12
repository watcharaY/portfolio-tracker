package postgres

import (
	"testing"
	"testing/fstest"
)

func TestEmbeddedMigrationsParse(t *testing.T) {
	ms, err := parseMigrations(migrationsFS)
	if err != nil {
		t.Fatal(err)
	}
	if ms[0].name != "20261012000000_init.sql" || ms[0].version != 20261012000000 {
		t.Fatalf("first migration = %+v, want 20261012000000_init.sql", ms[0])
	}
}

func TestParseMigrationsOrdersByVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/20261101000000_b.sql": &fstest.MapFile{},
		"migrations/20261012000000_a.sql": &fstest.MapFile{},
	}
	ms, err := parseMigrations(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].version != 20261012000000 || ms[1].version != 20261101000000 {
		t.Fatalf("parseMigrations = %+v, want 20261012000000 then 20261101000000", ms)
	}
}

func TestParseMigrationsRejects(t *testing.T) {
	for name, files := range map[string][]string{
		"sequence prefix": {"001_init.sql"},
		"not a timestamp": {"20261399000000_bad_month.sql"},
		"uppercase name":  {"20261012000000_Init.sql"},
		"duplicate":       {"20261012000000_a.sql", "20261012000000_b.sql"},
	} {
		fsys := fstest.MapFS{}
		for _, f := range files {
			fsys["migrations/"+f] = &fstest.MapFile{}
		}
		if _, err := parseMigrations(fsys); err == nil {
			t.Errorf("%s: parseMigrations(%v) succeeded, want error", name, files)
		}
	}
}

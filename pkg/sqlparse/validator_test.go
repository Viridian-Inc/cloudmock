package sqlparse

import (
	"testing"
)

func TestParse_Select(t *testing.T) {
	r := Parse("SELECT id, name FROM users WHERE active = true")
	if r.StatementType != "SELECT" {
		t.Errorf("expected SELECT, got %s", r.StatementType)
	}
	if len(r.Tables) != 1 || r.Tables[0] != "users" {
		t.Errorf("expected [users], got %v", r.Tables)
	}
	if len(r.Columns) != 2 {
		t.Errorf("expected 2 columns, got %v", r.Columns)
	}
}

func TestParse_SelectStar(t *testing.T) {
	r := Parse("SELECT * FROM orders")
	if len(r.Columns) != 0 {
		t.Errorf("expected 0 columns for *, got %v", r.Columns)
	}
	if len(r.Tables) != 1 || r.Tables[0] != "orders" {
		t.Errorf("expected [orders], got %v", r.Tables)
	}
}

func TestParse_Join(t *testing.T) {
	r := Parse("SELECT u.id FROM users u JOIN orders o ON u.id = o.user_id")
	if len(r.Tables) != 2 {
		t.Errorf("expected 2 tables, got %v", r.Tables)
	}
}

func TestParse_Insert(t *testing.T) {
	r := Parse("INSERT INTO logs (msg) VALUES ('hello')")
	if r.StatementType != "INSERT" {
		t.Errorf("expected INSERT, got %s", r.StatementType)
	}
	if len(r.Tables) != 1 || r.Tables[0] != "logs" {
		t.Errorf("expected [logs], got %v", r.Tables)
	}
}

func TestParse_CreateTable(t *testing.T) {
	r := Parse("CREATE TABLE IF NOT EXISTS metrics (id INT, value FLOAT)")
	if r.StatementType != "CREATE" {
		t.Errorf("expected CREATE, got %s", r.StatementType)
	}
	if len(r.Tables) != 1 || r.Tables[0] != "metrics" {
		t.Errorf("expected [metrics], got %v", r.Tables)
	}
}

func TestParse_EmptySQL(t *testing.T) {
	r := Parse("")
	if r.IsValid {
		t.Error("expected invalid for empty SQL")
	}
}

func TestParse_UnrecognizedType(t *testing.T) {
	r := Parse("GRANT SELECT ON users TO admin")
	if r.IsValid {
		t.Error("expected invalid for unrecognized type")
	}
}

func TestParse_Comments(t *testing.T) {
	r := Parse("-- get users\nSELECT id FROM users /* active only */ WHERE active = 1")
	if r.StatementType != "SELECT" {
		t.Errorf("expected SELECT, got %s", r.StatementType)
	}
	if len(r.Tables) != 1 || r.Tables[0] != "users" {
		t.Errorf("expected [users], got %v", r.Tables)
	}
}

func TestValidate_TableNotFound(t *testing.T) {
	reg := NewSchemaRegistry()
	reg.Register("default", "users", []string{"id", "name"})

	parsed := Parse("SELECT id FROM missing_table")
	errs := Validate(parsed, reg)
	if len(errs) == 0 {
		t.Error("expected error for missing table")
	}
}

func TestValidate_ColumnNotFound(t *testing.T) {
	reg := NewSchemaRegistry()
	reg.Register("default", "users", []string{"id", "name"})

	parsed := Parse("SELECT nonexistent FROM users")
	errs := Validate(parsed, reg)
	if len(errs) == 0 {
		t.Error("expected error for missing column")
	}
}

func TestValidate_Valid(t *testing.T) {
	reg := NewSchemaRegistry()
	reg.Register("default", "users", []string{"id", "name", "active"})

	parsed := Parse("SELECT id, name FROM users")
	errs := Validate(parsed, reg)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestParse_MultipleFromTables(t *testing.T) {
	r := Parse("SELECT a.id, b.name FROM users a, orders b WHERE a.id = b.user_id")
	if len(r.Tables) < 2 {
		t.Errorf("expected 2+ tables, got %v", r.Tables)
	}
}

func TestParse_Drop(t *testing.T) {
	r := Parse("DROP TABLE IF EXISTS temp_data")
	if r.StatementType != "DROP" {
		t.Errorf("expected DROP, got %s", r.StatementType)
	}
	if len(r.Tables) != 1 || r.Tables[0] != "temp_data" {
		t.Errorf("expected [temp_data], got %v", r.Tables)
	}
}

// Package sqlparse provides a lightweight SQL validator for Athena and Redshift.
// It extracts table and column references from SQL statements and validates them
// against a schema registry. It does NOT execute queries.
package sqlparse

import (
	"fmt"
	"regexp"
	"strings"
)

// Schema describes a table's columns for validation.
type Schema struct {
	Database string
	Table    string
	Columns  []string
}

// SchemaRegistry maps "database.table" to column lists.
type SchemaRegistry struct {
	schemas map[string]*Schema
}

// NewSchemaRegistry creates an empty registry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{schemas: make(map[string]*Schema)}
}

// Register adds a table schema.
func (r *SchemaRegistry) Register(database, table string, columns []string) {
	key := database + "." + table
	r.schemas[key] = &Schema{Database: database, Table: table, Columns: columns}
}

// Len returns the number of registered schemas.
func (r *SchemaRegistry) Len() int {
	return len(r.schemas)
}

// Lookup returns the schema for a table, checking both "db.table" and just "table".
func (r *SchemaRegistry) Lookup(ref string) (*Schema, bool) {
	if s, ok := r.schemas[ref]; ok {
		return s, true
	}
	// Try matching just the table name against all schemas.
	for _, s := range r.schemas {
		if s.Table == ref {
			return s, true
		}
	}
	return nil, false
}

// ParseResult contains the extracted elements from a SQL statement.
type ParseResult struct {
	StatementType string   // SELECT, INSERT, CREATE, etc.
	Tables        []string // Table references from FROM/JOIN
	Columns       []string // Column references from SELECT/WHERE
	IsValid       bool
	Errors        []string
}

var (
	reFrom    = regexp.MustCompile(`(?i)\bFROM\s+((?:[a-zA-Z_][\w.]*(?:\s+(?:AS\s+)?[a-zA-Z_]\w*)?\s*,\s*)*[a-zA-Z_][\w.]*(?:\s+(?:AS\s+)?[a-zA-Z_]\w*)?)`)
	reJoin    = regexp.MustCompile(`(?i)\bJOIN\s+([a-zA-Z_][\w.]*)`)
	reSelect  = regexp.MustCompile(`(?i)^SELECT\s+(.*?)\s+FROM\b`)
	reInsert  = regexp.MustCompile(`(?i)^INSERT\s+INTO\s+([a-zA-Z_][\w.]*)`)
	reCreate  = regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][\w.]*)`)
	reDrop    = regexp.MustCompile(`(?i)^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?([a-zA-Z_][\w.]*)`)
	reComment = regexp.MustCompile(`--[^\n]*|/\*[\s\S]*?\*/`)
)

// Parse extracts table and column references from a SQL statement.
func Parse(sql string) *ParseResult {
	result := &ParseResult{IsValid: true}

	// Strip comments.
	cleaned := reComment.ReplaceAllString(sql, " ")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "empty SQL statement")
		return result
	}

	// Detect statement type.
	upper := strings.ToUpper(cleaned)
	switch {
	case strings.HasPrefix(upper, "SELECT"):
		result.StatementType = "SELECT"
	case strings.HasPrefix(upper, "INSERT"):
		result.StatementType = "INSERT"
	case strings.HasPrefix(upper, "UPDATE"):
		result.StatementType = "UPDATE"
	case strings.HasPrefix(upper, "DELETE"):
		result.StatementType = "DELETE"
	case strings.HasPrefix(upper, "CREATE"):
		result.StatementType = "CREATE"
	case strings.HasPrefix(upper, "DROP"):
		result.StatementType = "DROP"
	case strings.HasPrefix(upper, "ALTER"):
		result.StatementType = "ALTER"
	case strings.HasPrefix(upper, "SHOW"):
		result.StatementType = "SHOW"
	case strings.HasPrefix(upper, "DESCRIBE"):
		result.StatementType = "DESCRIBE"
	default:
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("unrecognized SQL statement type: %.20s", cleaned))
		return result
	}

	// Extract table references.
	if m := reFrom.FindStringSubmatch(cleaned); len(m) > 1 {
		for _, t := range strings.Split(m[1], ",") {
			t = strings.TrimSpace(t)
			// Remove alias (e.g., "table AS t" or "table t")
			parts := strings.Fields(t)
			if len(parts) > 0 {
				result.Tables = append(result.Tables, parts[0])
			}
		}
	}
	for _, m := range reJoin.FindAllStringSubmatch(cleaned, -1) {
		if len(m) > 1 {
			result.Tables = append(result.Tables, m[1])
		}
	}
	if m := reInsert.FindStringSubmatch(cleaned); len(m) > 1 {
		result.Tables = append(result.Tables, m[1])
	}
	if m := reCreate.FindStringSubmatch(cleaned); len(m) > 1 {
		result.Tables = append(result.Tables, m[1])
	}
	if m := reDrop.FindStringSubmatch(cleaned); len(m) > 1 {
		result.Tables = append(result.Tables, m[1])
	}

	// Extract column references from SELECT clause.
	if result.StatementType == "SELECT" {
		if m := reSelect.FindStringSubmatch(cleaned); len(m) > 1 {
			cols := m[1]
			if strings.TrimSpace(cols) != "*" {
				for _, c := range strings.Split(cols, ",") {
					c = strings.TrimSpace(c)
					// Remove alias
					parts := strings.Fields(c)
					if len(parts) > 0 {
						col := parts[0]
						// Remove table prefix (e.g., "t.column")
						if idx := strings.LastIndex(col, "."); idx >= 0 {
							col = col[idx+1:]
						}
						// Skip functions like COUNT(*), SUM(x)
						if !strings.Contains(col, "(") {
							result.Columns = append(result.Columns, col)
						}
					}
				}
			}
		}
	}

	return result
}

// Validate checks a parsed SQL result against a schema registry.
// Returns validation errors (table not found, column not found).
func Validate(parsed *ParseResult, registry *SchemaRegistry) []string {
	if registry == nil || !parsed.IsValid {
		return parsed.Errors
	}

	var errs []string

	// Validate tables exist.
	tableSchemas := make(map[string]*Schema)
	for _, tbl := range parsed.Tables {
		s, ok := registry.Lookup(tbl)
		if !ok {
			errs = append(errs, fmt.Sprintf("table not found: %s", tbl))
		} else {
			tableSchemas[tbl] = s
		}
	}

	// Validate columns exist in referenced tables.
	for _, col := range parsed.Columns {
		found := false
		for _, s := range tableSchemas {
			for _, sc := range s.Columns {
				if strings.EqualFold(sc, col) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found && len(tableSchemas) > 0 {
			errs = append(errs, fmt.Sprintf("column not found: %s", col))
		}
	}

	return errs
}

package util

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"time"
)

// CSVRow is a single row of a CSV export. Fixed holds the values for the
// ordered fixedKeys; Data holds dynamic key/value fields that may vary between
// rows (e.g. client-reported machine fields).
type CSVRow struct {
	Fixed []interface{}
	Data  map[string]interface{}
}

// WriteCSV writes a CSV document to w. The header is built from fixedKeys,
// followed by the union of all dynamic Data keys, sorted for deterministic
// output. A dynamic key that collides with a fixed column or an already-used
// column name is exported under a numeric-suffixed name (e.g. "ip" -> "ip1"),
// so no field is ever dropped. A UTF-8 BOM is prepended for Excel
// compatibility, and values that could be interpreted as spreadsheet formulas
// are escaped.
func WriteCSV(w io.Writer, fixedKeys []string, rows []CSVRow) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	if _, err := w.Write([]byte("\uFEFF")); err != nil {
		return err
	}

	used := make(map[string]bool, len(fixedKeys))
	cols := make([]string, 0, len(fixedKeys))
	for _, k := range fixedKeys {
		if used[k] {
			continue
		}
		used[k] = true
		cols = append(cols, k)
	}

	// Union of all dynamic data keys, sorted for deterministic column order.
	dataKeys := map[string]bool{}
	for _, r := range rows {
		for k := range r.Data {
			dataKeys[k] = true
		}
	}
	sorted := make([]string, 0, len(dataKeys))
	for k := range dataKeys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	dataCols := make(map[string]string, len(sorted))
	for _, k := range sorted {
		name := k
		for n := 1; used[name]; n++ {
			name = fmt.Sprintf("%s%d", k, n)
		}
		dataCols[k] = name
		used[name] = true
		cols = append(cols, name)
	}

	if err := cw.Write(cols); err != nil {
		return err
	}
	colIndex := make(map[string]int, len(cols))
	for i, k := range cols {
		colIndex[k] = i
	}
	for _, r := range rows {
		row := make([]string, len(cols))
		for i, v := range r.Fixed {
			if i >= len(cols) {
				break
			}
			row[i] = sanitizeCSVValue(formatCSVValue(v))
		}
		for k, name := range dataCols {
			if v, ok := r.Data[k]; ok {
				row[colIndex[name]] = sanitizeCSVValue(formatCSVValue(v))
			}
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// formatCSVValue converts a stored value to its CSV string representation.
func formatCSVValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case time.Time:
		return t.Format(time.RFC3339)
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

// sanitizeCSVValue prevents CSV formula injection by prefixing values that
// start with characters interpreted as formula indicators by spreadsheet
// applications (=, +, -, @, tab, carriage return).
func sanitizeCSVValue(v string) string {
	if len(v) > 0 {
		switch v[0] {
		case '=', '+', '-', '@', '\t', '\r':
			return "'" + v
		}
	}
	return v
}

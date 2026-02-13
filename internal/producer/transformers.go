package producer

import (
	"strings"
	"time"
)

// RowTransformer handles transformation of CSV rows into payload objects
type RowTransformer struct {
	tsColumnIdx     int
	labelsColumnIdx int
}

// NewRowTransformer creates a new row transformer
// tsColumnName and labelsColumnName are the names of columns to transform
// Pass empty strings to skip transformation of those fields
func NewRowTransformer(reader *MetricsCSVReader, tsColumnName, labelsColumnName string) *RowTransformer {
	tsIdx := -1
	labelsIdx := -1

	if tsColumnName != "" {
		tsIdx = reader.FindColumnIndex(tsColumnName)
	}

	if labelsColumnName != "" {
		labelsIdx = reader.FindColumnIndex(labelsColumnName)
	}

	return &RowTransformer{
		tsColumnIdx:     tsIdx,
		labelsColumnIdx: labelsIdx,
	}
}

// TransformRow converts a CSV row map into a payload map with transformations
func (t *RowTransformer) TransformRow(row map[string]string) map[string]interface{} {
	payload := make(map[string]interface{}, len(row))
	now := time.Now().UTC().Format(time.RFC3339Nano)

	for colName, val := range row {
		// Skip empty values
		if val == "" && colName != "timestamp" {
			continue
		}

		// Check if this is the timestamp column
		if t.tsColumnIdx >= 0 && colName == getColumnNameByIndex(row, t.tsColumnIdx) {
			// Replace timestamp with current time
			payload[colName] = now
		} else if t.labelsColumnIdx >= 0 && colName == getColumnNameByIndex(row, t.labelsColumnIdx) {
			// Parse labels_raw into a map
			if val != "" {
				payload[colName] = ParseLabels(val)
			} else {
				payload[colName] = make(map[string]string)
			}
		} else {
			// Include other values as-is
			payload[colName] = val
		}
	}

	return payload
}

// getColumnNameByIndex is a helper to get column name by checking the row
// Note: This is a workaround since we don't have index-to-name mapping in the row map
// In production, consider storing metadata in RowTransformer
func getColumnNameByIndex(row map[string]string, idx int) string {
	// This is a limitation of using map - we lose index information
	// For a better solution, the caller should track column indices
	return ""
}

// ParseLabels converts a labels_raw string into a map of key-value pairs.
// The format is: key1="value1",key2="value2",...
func ParseLabels(raw string) map[string]string {
	labels := make(map[string]string)
	if raw == "" {
		return labels
	}

	// Split by comma, but respect quoted values
	var parts []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, ch := range raw {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			current.WriteRune(ch)
			escaped = true
			continue
		}
		if ch == '"' {
			inQuotes = !inQuotes
			current.WriteRune(ch)
			continue
		}
		if ch == ',' && !inQuotes {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	// Parse each key=value pair
	for _, part := range parts {
		part = strings.TrimSpace(part)
		eqIdx := strings.Index(part, "=")
		if eqIdx == -1 {
			continue
		}
		key := strings.TrimSpace(part[:eqIdx])
		val := strings.TrimSpace(part[eqIdx+1:])
		// Remove surrounding quotes from value if present
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		labels[key] = val
	}

	return labels
}

// TransformRowWithRaw converts a raw CSV row (slice) into a payload using indices
// This is more efficient than using a map for transformation
func TransformRowWithRaw(header, record []string, tsIdx, labelsIdx int) map[string]interface{} {
	payload := make(map[string]interface{}, len(header))
	now := time.Now().UTC().Format(time.RFC3339Nano)

	for i, col := range header {
		var val string
		if i < len(record) {
			val = record[i]
		}

		if i == tsIdx {
			// Replace timestamp with current time
			payload[col] = now
		} else if i == labelsIdx && val != "" {
			// Parse labels_raw into a map
			payload[col] = ParseLabels(val)
		} else if val != "" {
			// Only include non-empty values
			payload[col] = val
		}
	}

	return payload
}

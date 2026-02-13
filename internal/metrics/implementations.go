package metrics

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// StringSorter implements the Sorter interface for sorting strings
type StringSorter struct{}

// NewStringSorter creates a new string sorter
func NewStringSorter() *StringSorter {
	return &StringSorter{}
}

// SortStrings sorts a slice of strings in ascending or descending order
func (s *StringSorter) SortStrings(items []string, order SortOrder) {
	if order == SortAscending {
		sort.Strings(items)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(items)))
	}
}

// TokenAuthenticator implements the Authenticator interface
type TokenAuthenticator struct {
	expectedToken string
}

// NewTokenAuthenticator creates a new token authenticator
func NewTokenAuthenticator(token string) *TokenAuthenticator {
	return &TokenAuthenticator{
		expectedToken: token,
	}
}

// Authenticate checks if the token matches the expected token
func (t *TokenAuthenticator) Authenticate(token string) bool {
	if t.expectedToken == "" {
		return true // no auth required
	}
	return token == t.expectedToken
}

// DefaultQueryParser implements the QueryParser interface
type DefaultQueryParser struct {
	params map[string][]string // Mimic http.Request.URL.Query() format
}

// NewDefaultQueryParser creates a new query parser from URL query values
func NewDefaultQueryParser(queryParams map[string][]string) *DefaultQueryParser {
	return &DefaultQueryParser{
		params: queryParams,
	}
}

// ParseInt parses an integer from query parameters with a default fallback
func (p *DefaultQueryParser) ParseInt(key string, defaultVal int) int {
	values, ok := p.params[key]
	if !ok || len(values) == 0 {
		return defaultVal
	}

	val := values[0]
	if n, err := strconv.Atoi(val); err == nil && n > 0 {
		return n
	}
	return defaultVal
}

// ParseSortOrder parses the sort order from query parameters
func (p *DefaultQueryParser) ParseSortOrder(key string, defaultVal SortOrder) SortOrder {
	values, ok := p.params[key]
	if !ok || len(values) == 0 {
		return defaultVal
	}

	order := strings.ToLower(values[0])
	if order == "desc" || order == "descending" {
		return SortDescending
	}
	return SortAscending
}

// ParseTimeParam parses a time parameter from query
// Supports RFC3339Nano and RFC3339 formats
func (p *DefaultQueryParser) ParseTimeParam(key string) (*time.Time, error) {
	values, ok := p.params[key]
	if !ok || len(values) == 0 {
		return nil, nil // not provided is not an error
	}

	val := values[0]

	// Try RFC3339Nano first
	if t, err := time.Parse(time.RFC3339Nano, val); err == nil {
		return &t, nil
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, val); err == nil {
		return &t, nil
	}

	return nil, ErrInvalidTime
}

// Pagination helper functions

// CalculateOffset calculates the starting index for pagination
func CalculateOffset(page, limit int) int {
	return (page - 1) * limit
}

// PaginateSlice returns a slice of the items based on pagination parameters
func PaginateSlice(items []string, offset, limit int) []string {
	if offset >= len(items) {
		return []string{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

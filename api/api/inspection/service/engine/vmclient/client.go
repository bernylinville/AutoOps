// Package vmclient provides VictoriaMetrics/Prometheus API client for inspection.
// Ported from inspection-tool/internal/client/vm/client.go.
package vmclient

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"dodevops-api/pkg/log"

	"github.com/go-resty/resty/v2"
)

// Client is a VictoriaMetrics/Prometheus API client.
type Client struct {
	httpClient *resty.Client
}

// NewClient creates a new VM client.
func NewClient(endpoint string, timeout int) *Client {
	if timeout <= 0 {
		timeout = 30
	}

	httpClient := resty.New().
		SetBaseURL(strings.TrimRight(endpoint, "/")).
		SetTimeout(time.Duration(timeout)*time.Second).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetRetryCount(3).
		SetRetryWaitTime(time.Second).
		SetRetryMaxWaitTime(8 * time.Second).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			if err != nil {
				return true
			}
			return resp != nil && resp.StatusCode() >= http.StatusInternalServerError
		})

	return &Client{httpClient: httpClient}
}

// Query executes an instant query.
func (c *Client) Query(ctx context.Context, query string) (*QueryResponse, error) {
	return c.QueryWithFilter(ctx, query, nil)
}

// QueryWithFilter executes instant query with optional host filtering.
// The filter is applied by injecting label matchers into the query.
func (c *Client) QueryWithFilter(ctx context.Context, query string, filter *HostFilter) (*QueryResponse, error) {
	finalQuery := query
	if filter != nil && !filter.IsEmpty() {
		finalQuery = c.injectLabelMatchers(query, filter)
	}

	log.Log().Debugf("[VM] executing query: %s", finalQuery)

	var result QueryResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("query", finalQuery).
		SetResult(&result).
		Get("/api/v1/query")

	if err != nil {
		log.Log().Errorf("[VM] query failed: %v", err)
		return nil, fmt.Errorf("VM query failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		log.Log().Errorf("[VM] status %d: %s", resp.StatusCode(), string(resp.Body()))
		return nil, fmt.Errorf("VM returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	if !result.IsSuccess() {
		log.Log().Errorf("[VM] API error [%s]: %s", result.ErrorType, result.Error)
		return nil, fmt.Errorf("VM API error [%s]: %s", result.ErrorType, result.Error)
	}

	if len(result.Warnings) > 0 {
		log.Log().Warnf("[VM] warnings: %v", result.Warnings)
	}

	log.Log().Debugf("[VM] query ok, %d results", len(result.Data.Result))
	return &result, nil
}

// QueryResults executes query and returns parsed results.
func (c *Client) QueryResults(ctx context.Context, query string) ([]QueryResult, error) {
	return c.QueryResultsWithFilter(ctx, query, nil)
}

// QueryResultsWithFilter executes query with filter and returns parsed results.
func (c *Client) QueryResultsWithFilter(ctx context.Context, query string, filter *HostFilter) ([]QueryResult, error) {
	resp, err := c.QueryWithFilter(ctx, query, filter)
	if err != nil {
		return nil, err
	}
	return ParseQueryResults(resp)
}

// QueryByIdent executes query and returns results grouped by ident.
func (c *Client) QueryByIdent(ctx context.Context, query string) (map[string]QueryResult, error) {
	return c.QueryByIdentWithFilter(ctx, query, nil)
}

// QueryByIdentWithFilter executes query with filter, grouped by ident.
func (c *Client) QueryByIdentWithFilter(ctx context.Context, query string, filter *HostFilter) (map[string]QueryResult, error) {
	results, err := c.QueryResultsWithFilter(ctx, query, filter)
	if err != nil {
		return nil, err
	}
	return GroupResultsByIdent(results), nil
}

// injectLabelMatchers injects label matchers into PromQL query.
// Business groups use OR (regex ~), tags use AND.
func (c *Client) injectLabelMatchers(query string, filter *HostFilter) string {
	if filter == nil || filter.IsEmpty() {
		return query
	}

	var matchers []string

	if len(filter.BusinessGroups) > 0 {
		escaped := make([]string, len(filter.BusinessGroups))
		for i, g := range filter.BusinessGroups {
			escaped[i] = escapeRegex(g)
		}
		matchers = append(matchers, fmt.Sprintf(`busigroup=~"%s"`, strings.Join(escaped, "|")))
	}

	for k, v := range filter.Tags {
		if !isValidLabelValue(v) {
			continue
		}
		matchers = append(matchers, fmt.Sprintf(`%s="%s"`, k, strings.ReplaceAll(v, `"`, `\"`)))
	}

	if len(matchers) == 0 {
		return query
	}

	return injectMatchersToQuery(query, matchers)
}

// promqlMetricPattern matches a PromQL metric name with optional label selector.
// Compiled once at package init to avoid per-call regex compilation.
var promqlMetricPattern = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)(\{[^}]*\})?`)

// injectMatchersToQuery injects label matchers into PromQL metric selectors.
// Skips identifiers followed by '(' (function calls) and identifiers inside
// range selectors [...] to avoid corrupting queries.
func injectMatchersToQuery(query string, matchers []string) string {
	if len(matchers) == 0 {
		return query
	}
	matcherStr := strings.Join(matchers, ", ")

	matches := promqlMetricPattern.FindAllStringSubmatchIndex(query, -1)
	if len(matches) == 0 {
		return query
	}

	var result strings.Builder
	lastEnd := 0
	for _, m := range matches {
		fullStart, fullEnd := m[0], m[1]
		nameStart, nameEnd := m[2], m[3]

		// Write text between matches.
		result.WriteString(query[lastEnd:fullStart])

		// Skip identifiers inside range selectors [...] or followed by '('.
		if isInsideBrackets(query, fullStart) || isFollowedByParen(query, fullEnd) {
			result.WriteString(query[fullStart:fullEnd])
			lastEnd = fullEnd
			continue
		}

		metricName := query[nameStart:nameEnd]
		hasBraces := m[4] != -1

		if hasBraces {
			existing := query[m[4]+1 : m[5]-1]
			if existing == "" {
				result.WriteString(metricName)
				result.WriteString("{")
				result.WriteString(matcherStr)
				result.WriteString("}")
			} else {
				result.WriteString(metricName)
				result.WriteString("{")
				result.WriteString(existing)
				result.WriteString(", ")
				result.WriteString(matcherStr)
				result.WriteString("}")
			}
		} else {
			result.WriteString(metricName)
			result.WriteString("{")
			result.WriteString(matcherStr)
			result.WriteString("}")
		}
		lastEnd = fullEnd
	}
	result.WriteString(query[lastEnd:])

	return result.String()
}

// isInsideBrackets returns true if position pos is inside square brackets [...].
func isInsideBrackets(query string, pos int) bool {
	depth := 0
	for i := 0; i < pos; i++ {
		switch query[i] {
		case '[':
			depth++
		case ']':
			depth--
		}
	}
	return depth > 0
}

// isFollowedByParen returns true if the position is followed by '(' (function call).
func isFollowedByParen(query string, pos int) bool {
	rest := strings.TrimLeft(query[pos:], " \t")
	return len(rest) > 0 && rest[0] == '('
}

// escapeRegex escapes special regex characters.
func escapeRegex(s string) string {
	special := []string{`\`, ".", "+", "*", "?", "^", "$", "(", ")", "[", "]", "{", "}", "|"}
	for _, char := range special {
		s = strings.ReplaceAll(s, char, `\`+char)
	}
	return s
}

// isValidLabelValue checks for PromQL metacharacters that could break query injection.
// Rejects characters: {}[]=~!, and backslash.
func isValidLabelValue(s string) bool {
	for _, c := range s {
		switch c {
		case '{', '}', '[', ']', '=', '~', '!', ',', '\\':
			return false
		}
	}
	return true
}

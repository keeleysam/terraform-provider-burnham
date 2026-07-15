package promql

import (
	"strings"
	"testing"
)

// validQueries are real PromQL expressions spanning the language (verbatim from
// Prometheus docs and real alerting rules).
var validQueries = []string{
	`up`,
	`http_requests_total{job="apiserver", handler="/api/comments"}`,
	`http_requests_total{environment=~"staging|testing", method!="GET"}`,
	`rate(http_requests_total[5m])`,
	`increase(http_requests_total{job="api"}[1h])`,
	`sum by (job) (rate(http_requests_total[5m]))`,
	`sum without (instance) (node_memory_MemFree_bytes)`,
	`topk(5, sum by (app) (rate(request_duration_seconds_count[5m])))`,
	`histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket[5m])))`,
	`100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`,
	`rate(errors_total[5m]) / rate(requests_total[5m]) > 0.05`,
	`http_requests_total offset 5m`,
	`http_requests_total @ 1609746000`,
	`max_over_time(rate(http_requests_total[5m])[30m:1m])`,
	`label_replace(up{job="api"}, "service", "$1", "job", "(.*)")`,
	`absent(up{job="critical-service"})`,
}

// invalidQueries are rejected by the parser (syntax or type errors).
var invalidQueries = []string{
	`sum(rate(http_requests_total[5m])`,      // unbalanced parenthesis
	`rate(http_requests_total)`,              // rate() needs a range vector
	`http_requests_total{job="api"`,          // unterminated matcher
	`sum by job (http_requests_total)`,       // by-clause needs parentheses
	`histogram_quantile("0.95", foo_bucket)`, // first arg must be scalar
	`http_requests_total[5m] + rate(x[5m])`,  // range vector in arithmetic
	`foo{bar=~"[unterminated}`,               // invalid regex matcher
	``,                                       // empty
}

func TestIsValid(t *testing.T) {
	for _, q := range validQueries {
		if !IsValid(q) {
			t.Errorf("IsValid(%q) = false, want true", q)
		}
	}
	for _, q := range invalidQueries {
		if IsValid(q) {
			t.Errorf("IsValid(%q) = true, want false", q)
		}
	}
}

func TestFormatCanonicalIdempotent(t *testing.T) {
	for _, q := range validQueries {
		f1, err := Format(q, false)
		if err != nil {
			t.Errorf("Format(%q): %v", q, err)
			continue
		}
		if !IsValid(f1) {
			t.Errorf("formatted output is not valid: %q", f1)
		}
		f2, err := Format(f1, false)
		if err != nil {
			t.Errorf("Format(Format(%q)): %v", q, err)
			continue
		}
		if f1 != f2 {
			t.Errorf("Format not idempotent for %q:\n %q\n %q", q, f1, f2)
		}
	}
}

func TestFormatNormalizesSpacing(t *testing.T) {
	messy, err := Format(`sum  (  rate( http_requests_total [5m] ) )`, false)
	if err != nil {
		t.Fatalf("Format messy: %v", err)
	}
	clean, err := Format(`sum(rate(http_requests_total[5m]))`, false)
	if err != nil {
		t.Fatalf("Format clean: %v", err)
	}
	if messy != clean {
		t.Errorf("spacing not normalized:\n %q\n %q", messy, clean)
	}
}

func TestFormatPretty(t *testing.T) {
	long := `sum by (job) (rate(http_requests_total{code=~"5.."}[5m])) / sum by (job) (rate(http_requests_total[5m])) > 0.05`
	got, err := Format(long, true)
	if err != nil {
		t.Fatalf("Format pretty: %v", err)
	}
	if !strings.Contains(got, "\n") {
		t.Errorf("expected multi-line pretty output, got %q", got)
	}
	if !IsValid(got) {
		t.Errorf("pretty output is not valid PromQL: %q", got)
	}
}

func TestFormatDropsComments(t *testing.T) {
	got, err := Format("up # trailing comment", false)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if strings.Contains(got, "#") {
		t.Errorf("expected comment dropped, got %q", got)
	}
}

func TestFormatInvalid(t *testing.T) {
	if _, err := Format(`sum(`, false); err == nil {
		t.Error("Format of invalid input should error")
	}
}

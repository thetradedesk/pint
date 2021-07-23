package checks_test

import (
	"testing"

	"github.com/cloudflare/pint/internal/checks"
)

func TestComparisonCheck(t *testing.T) {
	testCases := []checkTest{
		{
			description: "ignores recording rules",
			content:     "- record: foo\n  expr: up == 0\n",
			checker:     checks.NewComparisonCheck(checks.Bug),
		},
		{
			description: "ignores rules with syntax errors",
			content:     "- alert: Foo Is Down\n  expr: sum(\n",
			checker:     checks.NewComparisonCheck(checks.Bug),
		},
		{
			description: "alert expr with > condition",
			content:     "- alert: Foo Is Down\n  for: 10m\n  expr: up{job=\"foo\"} > 0\n",
			checker:     checks.NewComparisonCheck(checks.Bug),
		},
		{
			description: "alert expr with >= condition",
			content:     "- alert: Foo Is Down\n  for: 10m\n  expr: up{job=\"foo\"} >= 1\n",
			checker:     checks.NewComparisonCheck(checks.Bug)},
		{
			description: "alert expr with == condition",
			content:     "- alert: Foo Is Down\n  for: 10m\n  expr: up{job=\"foo\"} == 1\n",
			checker:     checks.NewComparisonCheck(checks.Bug)},
		{
			description: "alert expr without any condition",
			content:     "- alert: Foo Is Down\n  expr: up{job=\"foo\"}\n",
			checker:     checks.NewComparisonCheck(checks.Warning), problems: []checks.Problem{
				{
					Fragment: `up{job="foo"}`,
					Lines:    []int{2},
					Reporter: "alerts/count",
					Text:     "alert query doesn't have any condition, it will always fire if the metric exists",
					Severity: checks.Warning,
				},
			},
		},
	}

	runTests(t, testCases)
}
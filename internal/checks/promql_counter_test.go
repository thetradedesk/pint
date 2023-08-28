package checks_test

import (
	"fmt"
	"strings"
	"testing"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/cloudflare/pint/internal/checks"
	"github.com/cloudflare/pint/internal/promapi"
)

func newCounterCheck(prom *promapi.FailoverGroup) checks.RuleChecker {
	return checks.NewCounterCheck(prom)
}

func CounterMustUseFuncTextForAlert(name string) string {
	allowedFuncString := "`" + strings.Join(checks.AllowedCounterFuncsForAlerts, "`, `") + "`"
	return fmt.Sprintf("Counter metric `%s` should be used with one of following functions: (%s).", name, allowedFuncString)
}

func CounterMustUseFuncTextForRecordingRule(name string) string {
	allowedFuncString := "`" + strings.Join(checks.AllowedCounterFuncsForRecordingRules, "`, `") + "`"
	return fmt.Sprintf("Counter metric `%s` should be used with one of following functions: (%s).", name, allowedFuncString)
}

func TestCounterCheck(t *testing.T) {
	testCases := []checkTest{
		{
			description: "use counter with count",
			content:     "- record: foo\n  expr:  max(count(foo)) \n   ",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"foo": {{Type: "counter"}},
					}},
				},
			},
		},
		{
			description: "use counter on RHS of unless",
			content:     "- record: foo\n  expr:  foo unless on () bar  \n   ",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"bar": {{Type: "counter"}},
					}},
				},
			},
		},
		{
			description: "use counter on LHS of unless",
			content:     "- record: foo\n  expr:  foo unless on () bar  \n   ",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems: func(uri string) []checks.Problem {
				return []checks.Problem{
					{
						Fragment: "foo",
						Lines:    []int{2},
						Reporter: "promql/counter",
						Text:     CounterMustUseFuncTextForRecordingRule("foo"),
						Severity: checks.Warning,
					},
				}
			},
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"foo": {{Type: "counter"}},
					}},
				},
			},
		},
		{
			description: "use gauge without rate",
			content:     "- record: foo\n  expr: bar + bar + bar\n",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"bar": {{Type: "gauge"}},
					}},
				},
			},
		},
		{
			description: "use counter with rate",
			content:     "- record: foo\n  expr: irate(foo[1m])\n",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"foo": {{Type: "counter"}},
					}},
				},
			},
		},
		{
			description: "use counter with absent",
			content:     "- record: foo\n  expr: absent(foo) \n",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"foo": {{Type: "counter"}},
					}},
				},
			},
		},
		{
			description: "use counter with and without rate",
			content:     "- record: foo\n  expr: increase(foo[1m]) and sum(foo offset 1m)\n",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems: func(uri string) []checks.Problem {
				return []checks.Problem{
					{
						Fragment: "foo offset 1m",
						Lines:    []int{2},
						Reporter: "promql/counter",
						Text:     CounterMustUseFuncTextForRecordingRule("foo"),
						Severity: checks.Warning,
					},
				}
			},
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"foo": {{Type: "counter"}},
					}},
				},
			},
		},
		{
			description: "use counter with delta",
			content:     "- record: foo\n  expr: delta(foo[1m]) \n",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems: func(uri string) []checks.Problem {
				return []checks.Problem{
					{
						Fragment: "foo",
						Lines:    []int{2},
						Reporter: "promql/counter",
						Text:     CounterMustUseFuncTextForRecordingRule("foo"),
						Severity: checks.Warning,
					},
				}
			},
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"foo": {{Type: "counter"}},
					}},
				},
			},
		},
		{
			description: "use counter with irate in alert",
			content:     "- alert: my alert\n  expr: irate(foo[5m])\n",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems: func(uri string) []checks.Problem {
				return []checks.Problem{
					{
						Fragment: "foo",
						Lines:    []int{2},
						Reporter: "promql/counter",
						Text:     CounterMustUseFuncTextForAlert("foo"),
						Severity: checks.Warning,
					},
				}
			},
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp: metadataResponse{metadata: map[string][]v1.Metadata{
						"foo": {{Type: "counter"}},
					}},
				},
			},
		},
		{
			description: "empty data from Prometheus API",
			content:     "- alert: my alert\n  expr: irate(foo[5m])\n",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp:  metadataResponse{metadata: map[string][]v1.Metadata{}},
				},
			},
		},
		{
			description: "500 error from Prometheus API",
			content:     "- record: foo\n  expr: rate(foo[5m])\n",
			checker:     newCounterCheck,
			prometheus:  newSimpleProm,
			problems: func(uri string) []checks.Problem {
				return []checks.Problem{
					{
						Fragment: "foo",
						Lines:    []int{2},
						Reporter: "promql/counter",
						Text:     checkErrorUnableToRun(checks.CounterCheckName, "prom", uri, "server_error: internal error"),
						Severity: checks.Bug,
					},
				}
			},
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp:  respondWithInternalError(),
				},
			},
		},

		{
			description: "empty data from first Prometheus API - use counter with delta",
			content:     "- record: foo\n  expr: delta(foo[5m])\n",
			checker:     newCounterCheck,
			prometheus:  newDoubleProm,
			problems:    noProblems,
			mocks: []*prometheusMock{
				{
					conds: []requestCondition{requireMetadataPath},
					resp:  metadataResponse{metadata: map[string][]v1.Metadata{}},
				},
			},
		},
	}

	runTests(t, testCases)
}

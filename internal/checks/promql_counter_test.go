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

func CounterMustUseRateText(name string) string {
	allowedFuncString := "`" + strings.Join(checks.AllowedCounterFuncNames, "`, `") + "`"
	return fmt.Sprintf("Counter metric `%s` should be used with one of following functions: (%s).", name, allowedFuncString)
}

func TestCounterCheck(t *testing.T) {
	testCases := []checkTest{
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
			content:     "- record: foo\n  expr: rate(foo[1m])\n",
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
						Text:     CounterMustUseRateText("foo"),
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
						Text:     CounterMustUseRateText("foo"),
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
	}

	runTests(t, testCases)
}

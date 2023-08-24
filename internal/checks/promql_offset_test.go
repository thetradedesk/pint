package checks_test

import (
	"testing"
	"time"

	"github.com/cloudflare/pint/internal/checks"
	"github.com/cloudflare/pint/internal/promapi"
)

func newOffsetCheck(_ *promapi.FailoverGroup) checks.RuleChecker {
	dur, _ := time.ParseDuration("5m")
	return checks.NewOffsetCheck(checks.MustTemplatedRegexp("aws_.*"), dur)
}

func TestOffsetCheck(t *testing.T) {
	testCases := []checkTest{
		{
			description: "ignores rules with syntax errors",
			content:     "- alert: foo\n  expr: sum(foo) without(\n",
			checker:     newOffsetCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
		},
		{
			description: "ignores rules that don't have a prefix",
			content:     "- alert: foo\n  expr: foo)\n",
			checker:     newOffsetCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
		},
		{
			description: "check has no problems when offset is valid",
			content:     "- alert: foo\n  expr: aws_foo offset 5m\n",
			checker:     newOffsetCheck,
			prometheus:  newSimpleProm,
			problems:    noProblems,
		},
		{
			description: "check has problem when offset doesn't exist",
			content:     "- alert: foo\n  expr: aws_foo\n",
			checker:     newOffsetCheck,
			prometheus:  newSimpleProm,
			problems: func(_ string) []checks.Problem {
				return []checks.Problem{
					{
						Fragment: "aws_foo",
						Lines:    []int{2},
						Reporter: checks.OffsetCheckName,
						Text:     `the aws_foo metric requires a minimum offset of 5m0s`,
						Severity: checks.Warning,
					},
				}
			},
		},
		{
			description: "check has problem when offset is too low",
			content:     "- alert: foo\n  expr: aws_foo offset 2m\n",
			checker:     newOffsetCheck,
			prometheus:  newSimpleProm,
			problems: func(_ string) []checks.Problem {
				return []checks.Problem{
					{
						Fragment: "aws_foo offset 2m",
						Lines:    []int{2},
						Reporter: checks.OffsetCheckName,
						Text:     `the aws_foo offset 2m metric requires a minimum offset of 5m0s`,
						Severity: checks.Warning,
					},
				}
			},
		},
		{
			description: "check has problem when any metric is missing an offset",
			content:     "- alert: foo\n  expr: aws_foo offset 10m + aws_bar\n",
			checker:     newOffsetCheck,
			prometheus:  newSimpleProm,
			problems: func(_ string) []checks.Problem {
				return []checks.Problem{
					{
						Fragment: "aws_bar",
						Lines:    []int{2},
						Reporter: checks.OffsetCheckName,
						Text:     `the aws_bar metric requires a minimum offset of 5m0s`,
						Severity: checks.Warning,
					},
				}
			},
		},
	}
	runTests(t, testCases)
}

package checks

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promParser "github.com/prometheus/prometheus/promql/parser"

	"github.com/cloudflare/pint/internal/discovery"
	"github.com/cloudflare/pint/internal/parser"
	"github.com/cloudflare/pint/internal/promapi"
)

const (
	CounterCheckName = "promql/counter"
)

var (
	AllowedCounterFuncsForRecordingRules = []string{"rate", "irate", "increase", "absent", "absent_over_time", "count", "count_over_time", "present_over_time"}
	AllowedCounterFuncsForAlerts         = []string{"rate", "increase", "absent", "absent_over_time", "count", "count_over_time", "present_over_time"}
)

func NewCounterCheck(prom *promapi.FailoverGroup) CounterCheck {
	return CounterCheck{prom: prom}
}

type CounterCheck struct {
	prom *promapi.FailoverGroup
}

func (c CounterCheck) Meta() CheckMeta {
	return CheckMeta{IsOnline: true}
}

func (c CounterCheck) String() string {
	return fmt.Sprintf("%s(%s)", CounterCheckName, c.prom.Name())
}

func (c CounterCheck) Reporter() string {
	return CounterCheckName
}

func (c CounterCheck) Check(ctx context.Context, _ string, rule parser.Rule, entries []discovery.Entry) (problems []Problem) {
	expr := rule.Expr()

	if expr.SyntaxError != nil || (rule.AlertingRule == nil && rule.RecordingRule == nil) {
		return problems
	}

	isAlertRule := rule.AlertingRule != nil

	isCounterMap := &IsCounterMapForCounterCheck{
		values: make(map[string]bool),
	}
	for _, problem := range c.checkNode(ctx, expr.Query, entries, false, isAlertRule, isCounterMap) {
		problems = append(problems, Problem{
			Fragment: problem.expr,
			Lines:    expr.Lines(),
			Reporter: c.Reporter(),
			Text:     problem.text,
			Severity: problem.severity,
		})
	}

	return problems
}

func (c CounterCheck) checkNode(ctx context.Context, node *parser.PromQLNode, entries []discovery.Entry, parentUsesAllowedFunction, isAlertRule bool, isCounterMap *IsCounterMapForCounterCheck) (problems []exprProblem) {
	allowedFuncs := AllowedCounterFuncsForRecordingRules
	if isAlertRule {
		allowedFuncs = AllowedCounterFuncsForAlerts
	}

	if s, ok := node.Node.(*promParser.VectorSelector); ok {
		isCounter, ok := isCounterMap.values[s.Name]
		if ok {
			if !isCounter {
				return problems
			}
		} else {
			metadata, err := c.prom.Metadata(ctx, s.Name)
			if err != nil {
				text, severity := textAndSeverityFromError(err, c.Reporter(), c.prom.Name(), Bug)
				problems = append(problems, exprProblem{
					expr:     s.Name,
					text:     text,
					severity: severity,
				})
				return problems
			}

			isCounter := false

			for _, m := range metadata.Metadata {
				if m.Type == v1.MetricTypeCounter {
					isCounter = true
					break // exit the loop as soon as you find a counter
				}
			}

			isCounterMap.values[s.Name] = isCounter
		}
		if isCounterMap.values[s.Name] && !parentUsesAllowedFunction {
			allowedFuncString := "`" + strings.Join(allowedFuncs, "`, `") + "`"

			p := exprProblem{
				expr: node.Expr,
				text: fmt.Sprintf("Counter metric `%s` should be used with one of following functions: (%s).", s.Name, allowedFuncString),
				// There can be valid edge cases like a recording rule: `foo{label="value"}` or being constrained to use a counter as an info metric for joining.
				severity: Warning,
			}
			problems = append(problems, p)
		}
	}

	// Matrix wraps a single vector, we will retain `parentUsesAllowedFunction` value. (e.g. rate(x) or rate(x[2m]) are treated equally)
	if _, ok := node.Node.(*promParser.MatrixSelector); !ok {
		parentUsesAllowedFunction = false

		if n, ok := node.Node.(*promParser.Call); ok && contains(allowedFuncs, n.Func.Name) {
			parentUsesAllowedFunction = true
		}
	}

	for _, child := range node.Children {
		problems = append(problems, c.checkNode(ctx, child, entries, parentUsesAllowedFunction, isAlertRule, isCounterMap)...)
	}

	return problems
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

type IsCounterMapForCounterCheck struct {
	values map[string]bool
}

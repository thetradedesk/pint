package checks

import (
	"context"
	"fmt"
	"github.com/cloudflare/pint/internal/discovery"
	"github.com/cloudflare/pint/internal/parser"
	"github.com/cloudflare/pint/internal/promapi"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promParser "github.com/prometheus/prometheus/promql/parser"
)

const (
	CounterCheckName = "promql/counter"
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
	return CounterCheckName
}

func (c CounterCheck) Reporter() string {
	return CounterCheckName
}

func (c CounterCheck) Check(ctx context.Context, _ string, rule parser.Rule, entries []discovery.Entry) (problems []Problem) {
	expr := rule.Expr()

	if expr.SyntaxError != nil {
		return problems
	}

	isCounterMap := &IsCounterMapForCounterCheck{
		values: make(map[string]bool),
	}
	for _, problem := range c.checkNode(ctx, expr.Query, entries, false, isCounterMap) {
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

func (c CounterCheck) checkNode(ctx context.Context, node *parser.PromQLNode, entries []discovery.Entry, parentUsesRate bool, isCounterMap *IsCounterMapForCounterCheck) (problems []exprProblem) {

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

		if isCounterMap.values[s.Name] && !parentUsesRate {
			p := exprProblem{
				expr:     node.Expr,
				text:     fmt.Sprintf("counter metric `%s` should be used with `rate`, `irate` or `increase` ", s.Name),
				severity: Warning,
			}
			problems = append(problems, p)
		}
	}
	if _, ok := node.Node.(*promParser.MatrixSelector); ok {
		// Matrix wraps a single vector, we will retain `parentUsesRate` value. (e.g. rate(x) or rate(x[2m]) are treated equally)
	} else {
		parentUsesRate = false
		if n, ok := node.Node.(*promParser.Call); ok && (n.Func.Name == "rate" || n.Func.Name == "irate" || n.Func.Name == "increase" || n.Func.Name == "absent" || n.Func.Name == "absent_over_time") {
			parentUsesRate = true
		}
	}

	for _, child := range node.Children {
		problems = append(problems, c.checkNode(ctx, child, entries, parentUsesRate, isCounterMap)...)
	}

	return problems
}

type IsCounterMapForCounterCheck struct {
	values map[string]bool
}

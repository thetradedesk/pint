package checks

import (
	"context"
	"fmt"
	"regexp"
	"time"

	promParser "github.com/prometheus/prometheus/promql/parser"

	"github.com/cloudflare/pint/internal/discovery"
	"github.com/cloudflare/pint/internal/parser"
)

const (
	OffsetCheckName = "promql/offset"
)

func NewOffsetCheck(prefix *TemplatedRegexp, minimumOffset time.Duration, severity Severity) OffsetCheck {
	return OffsetCheck{prefixRegex: prefix, min: minimumOffset, severity: severity}
}

type OffsetCheck struct {
	prefixRegex *TemplatedRegexp
	min         time.Duration
	severity    Severity
}

func (c OffsetCheck) Meta() CheckMeta {
	return CheckMeta{IsOnline: false}
}

func (c OffsetCheck) String() string {
	return fmt.Sprintf("%s(%s)", OffsetCheckName, c.prefixRegex.anchored)
}

func (c OffsetCheck) Reporter() string {
	return OffsetCheckName
}

func (c OffsetCheck) Check(_ context.Context, _ string, rule parser.Rule, _ []discovery.Entry) (problems []Problem) {
	expr := rule.Expr()

	if expr.SyntaxError != nil || (rule.AlertingRule == nil && rule.RecordingRule == nil) {
		return problems
	}

	for _, selector := range getSelectorsWithOffset(expr.Query) {
		match, _ := regexp.MatchString(c.prefixRegex.anchored, selector.Name)
		if match && c.min > selector.OriginalOffset {
			problems = append(problems, Problem{
				Fragment: selector.String(),
				Lines:    expr.Lines(),
				Reporter: c.Reporter(),
				Text:     fmt.Sprintf("the %s metric requires a minimum offset of %s", selector.String(), c.min.String()),
				Severity: c.severity,
			})
		}

	}

	return problems
}

func getSelectorsWithOffset(n *parser.PromQLNode) (selectors []promParser.VectorSelector) {
	if node, ok := n.Node.(*promParser.VectorSelector); ok {
		nc := promParser.VectorSelector{
			Name:           node.Name,
			LabelMatchers:  node.LabelMatchers,
			OriginalOffset: node.OriginalOffset,
		}
		selectors = append(selectors, nc)
	}

	for _, child := range n.Children {
		selectors = append(selectors, getSelectorsWithOffset(child)...)
	}

	return selectors
}

package checks

import (
	"context"
	"fmt"
	promParser "github.com/prometheus/prometheus/promql/parser"
	"regexp"
	"time"

	"github.com/cloudflare/pint/internal/discovery"
	"github.com/cloudflare/pint/internal/parser"
)

const (
	OffsetCheckName = "promql/offset"
)

func NewOffsetCheck(prefix *TemplatedRegexp, minimumOffset time.Duration) OffsetCheck {
	return OffsetCheck{prefixRegex: prefix, min: minimumOffset}
}

type OffsetCheck struct {
	prefixRegex *TemplatedRegexp
	min         time.Duration
}

func (c OffsetCheck) Meta() CheckMeta {
	return CheckMeta{IsOnline: false}
}

func (c OffsetCheck) String() string {
	return OffsetCheckName
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
				Severity: Warning,
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

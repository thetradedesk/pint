package config

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"

	"github.com/cloudflare/pint/internal/checks"
	"github.com/cloudflare/pint/internal/parser"
	"github.com/cloudflare/pint/internal/promapi"
)

type Rule struct {
	Match      []Match              `hcl:"match,block" json:"match,omitempty"`
	Ignore     []Match              `hcl:"ignore,block" json:"ignore,omitempty"`
	Aggregate  []AggregateSettings  `hcl:"aggregate,block" json:"aggregate,omitempty"`
	Annotation []AnnotationSettings `hcl:"annotation,block" json:"annotation,omitempty"`
	Label      []AnnotationSettings `hcl:"label,block" json:"label,omitempty"`
	Cost       *CostSettings        `hcl:"cost,block" json:"cost,omitempty"`
	Alerts     *AlertsSettings      `hcl:"alerts,block" json:"alerts,omitempty"`
	For        *ForSettings         `hcl:"for,block" json:"for,omitempty"`
	Reject     []RejectSettings     `hcl:"reject,block" json:"reject,omitempty"`
	RuleLink   []RuleLinkSettings   `hcl:"link,block" json:"link,omitempty"`
	Offset     []OffsetSettings     `hcl:"offset,block" json:"offset,omitempty"`
}

func (rule Rule) validate() (err error) {
	for _, match := range rule.Match {
		if err = match.validate(true); err != nil {
			return err
		}
	}

	for _, ignore := range rule.Ignore {
		if err = ignore.validate(false); err != nil {
			return err
		}
	}

	for _, aggr := range rule.Aggregate {
		if err = aggr.validate(); err != nil {
			return err
		}
	}

	for _, ann := range rule.Annotation {
		if err = ann.validate(); err != nil {
			return err
		}
	}

	for _, lab := range rule.Label {
		if err = lab.validate(); err != nil {
			return err
		}
	}

	if rule.Cost != nil {
		if err = rule.Cost.validate(); err != nil {
			return err
		}
	}

	if rule.Alerts != nil {
		if err = rule.Alerts.validate(); err != nil {
			return err
		}
	}

	for _, reject := range rule.Reject {
		if err = reject.validate(); err != nil {
			return err
		}
	}

	for _, link := range rule.RuleLink {
		if err = link.validate(); err != nil {
			return err
		}
	}

	if rule.For != nil {
		if err = rule.For.validate(); err != nil {
			return err
		}
	}

	for _, offset := range rule.Offset {
		if err = offset.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (rule Rule) resolveChecks(ctx context.Context, path string, r parser.Rule, prometheusServers []*promapi.FailoverGroup) []checkMeta {
	enabled := []checkMeta{}

	for _, ignore := range rule.Ignore {
		if ignore.IsMatch(ctx, path, r) {
			return enabled
		}
	}

	if len(rule.Match) > 0 {
		var found bool
		for _, match := range rule.Match {
			if match.IsMatch(ctx, path, r) {
				found = true
				break
			}
		}
		if !found {
			return enabled
		}
	}

	if len(rule.Aggregate) > 0 {
		var nameRegex *checks.TemplatedRegexp
		for _, aggr := range rule.Aggregate {
			if aggr.Name != "" {
				nameRegex = checks.MustTemplatedRegexp(aggr.Name)
			}
			severity := aggr.getSeverity(checks.Warning)
			for _, label := range aggr.Keep {
				enabled = append(enabled, checkMeta{
					name:  checks.AggregationCheckName,
					check: checks.NewAggregationCheck(nameRegex, label, true, severity),
				})
			}
			for _, label := range aggr.Strip {
				enabled = append(enabled, checkMeta{
					name:  checks.AggregationCheckName,
					check: checks.NewAggregationCheck(nameRegex, label, false, severity),
				})
			}
		}
	}

	if rule.Cost != nil {
		severity := rule.Cost.getSeverity(checks.Bug)
		evalDur, _ := parseDuration(rule.Cost.MaxEvaluationDuration)
		for _, prom := range prometheusServers {
			enabled = append(enabled, checkMeta{
				name:  checks.CostCheckName,
				check: checks.NewCostCheck(prom, rule.Cost.MaxSeries, rule.Cost.MaxTotalSamples, rule.Cost.MaxPeakSamples, evalDur, severity),
				tags:  prom.Tags(),
			})
		}
	}

	if len(rule.Annotation) > 0 {
		for _, ann := range rule.Annotation {
			var valueRegex *checks.TemplatedRegexp
			if ann.Value != "" {
				valueRegex = checks.MustTemplatedRegexp(ann.Value)
			}
			severity := ann.getSeverity(checks.Warning)
			enabled = append(enabled, checkMeta{
				name:  checks.AnnotationCheckName,
				check: checks.NewAnnotationCheck(checks.MustTemplatedRegexp(ann.Key), valueRegex, ann.Required, severity),
			})
		}
	}

	if len(rule.Label) > 0 {
		for _, lab := range rule.Label {
			var valueRegex *checks.TemplatedRegexp
			if lab.Value != "" {
				valueRegex = checks.MustTemplatedRegexp(lab.Value)
			}
			severity := lab.getSeverity(checks.Warning)
			enabled = append(enabled, checkMeta{
				name:  checks.LabelCheckName,
				check: checks.NewLabelCheck(lab.Key, valueRegex, lab.Required, severity),
			})
		}
	}

	if rule.Alerts != nil {
		qRange := time.Hour * 24
		if rule.Alerts.Range != "" {
			qRange, _ = parseDuration(rule.Alerts.Range)
		}
		qStep := time.Minute
		if rule.Alerts.Step != "" {
			qStep, _ = parseDuration(rule.Alerts.Step)
		}
		qResolve := time.Minute * 5
		if rule.Alerts.Resolve != "" {
			qResolve, _ = parseDuration(rule.Alerts.Resolve)
		}
		severity := rule.Alerts.getSeverity(checks.Information)
		for _, prom := range prometheusServers {
			enabled = append(enabled, checkMeta{
				name:  checks.AlertsCheckName,
				check: checks.NewAlertsCheck(prom, qRange, qStep, qResolve, rule.Alerts.MinCount, severity),
				tags:  prom.Tags(),
			})
		}
	}

	if len(rule.Reject) > 0 {
		for _, reject := range rule.Reject {
			severity := reject.getSeverity(checks.Bug)
			re := checks.MustTemplatedRegexp(reject.Regex)
			if reject.LabelKeys {
				enabled = append(enabled, checkMeta{
					name:  checks.RejectCheckName,
					check: checks.NewRejectCheck(true, false, re, nil, severity),
				})
			}
			if reject.LabelValues {
				enabled = append(enabled, checkMeta{
					name:  checks.RejectCheckName,
					check: checks.NewRejectCheck(true, false, nil, re, severity),
				})
			}
			if reject.AnnotationKeys {
				enabled = append(enabled, checkMeta{
					name:  checks.RejectCheckName,
					check: checks.NewRejectCheck(false, true, re, nil, severity),
				})
			}
			if reject.AnnotationValues {
				enabled = append(enabled, checkMeta{
					name:  checks.RejectCheckName,
					check: checks.NewRejectCheck(false, true, nil, re, severity),
				})
			}
		}
	}

	for _, link := range rule.RuleLink {
		severity := link.getSeverity(checks.Bug)
		re := checks.MustTemplatedRegexp(link.Regex)
		var timeout time.Duration
		if link.Timeout != "" {
			timeout, _ = parseDuration(link.Timeout)
		} else {
			timeout = time.Minute
		}
		enabled = append(enabled, checkMeta{
			name:  checks.RuleLinkCheckName,
			check: checks.NewRuleLinkCheck(re, link.URI, timeout, link.Headers, severity),
		})
	}

	if rule.For != nil {
		severity := rule.For.getSeverity(checks.Bug)
		var minFor, maxFor time.Duration
		if rule.For.Min != "" {
			minFor, _ = parseDuration(rule.For.Min)
		}
		if rule.For.Max != "" {
			maxFor, _ = parseDuration(rule.For.Max)
		}
		enabled = append(enabled, checkMeta{
			name:  checks.RuleForCheckName,
			check: checks.NewRuleForCheck(minFor, maxFor, severity),
		})
	}

	if len(rule.Offset) > 0 {
		for _, offset := range rule.Offset {
			prefix := checks.MustTemplatedRegexp(offset.Prefix)
			minOffset, _ := parseDuration(offset.Min)
			severity := offset.getSeverity(checks.Warning)
			enabled = append(enabled, checkMeta{
				name:  checks.OffsetCheckName,
				check: checks.NewOffsetCheck(prefix, minOffset, severity),
			})
		}
	}

	return enabled
}

func isEnabled(enabledChecks, disabledChecks []string, rule parser.Rule, name string, check checks.RuleChecker, promTags []string) bool {
	instance := check.String()

	comments := []string{
		fmt.Sprintf("disable %s", name),
		fmt.Sprintf("disable %s", instance),
	}
	for _, tag := range promTags {
		comments = append(comments, fmt.Sprintf("disable %s(+%s)", name, tag))
	}

	for _, comment := range comments {
		if rule.HasComment(comment) {
			log.Debug().
				Str("check", instance).
				Str("comment", comment).
				Msg("Check disabled by comment")
			return false
		}
	}

	disabled := []string{name, instance}
	for _, tag := range promTags {
		disabled = append(disabled, fmt.Sprintf("%s(+%s)", name, tag))
	}
	for _, comment := range rule.GetComments("snooze") {
		s := parser.ParseSnooze(comment.Value)
		if s == nil {
			continue
		}
		if !slices.Contains(disabled, s.Text) {
			continue
		}
		log.Debug().
			Str("check", instance).
			Str("comment", comment.String()).
			Time("until", s.Until).
			Str("snooze", s.Text).
			Msg("Check snoozed by comment")
		return false
	}

	for _, c := range disabledChecks {
		if c == name || c == instance {
			return false
		}
		for _, tag := range promTags {
			if c == fmt.Sprintf("%s(+%s)", name, tag) {
				return false
			}
		}
	}
	if len(enabledChecks) == 0 {
		return true
	}
	for _, c := range enabledChecks {
		if c == name {
			return true
		}
	}
	return false
}

func strictRegex(s string) *regexp.Regexp {
	return regexp.MustCompile("^" + s + "$")
}

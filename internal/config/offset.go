package config

import (
	"errors"
	"regexp"

	"github.com/cloudflare/pint/internal/checks"
)

type OffsetSettings struct {
	Prefix   string `hcl:"prefix" json:"regex,omitempty"`
	Min      string `hcl:"min" json:"min,omitempty"`
	Severity string `hcl:"severity,optional" json:"severity,omitempty"`
}

func (s OffsetSettings) validate() error {
	if s.Prefix == "" {
		return errors.New("offset prefix cannot be empty")
	}

	if s.Min == "" {
		return errors.New("minimum offset cannot be empty")
	}

	if _, err := regexp.Compile(s.Prefix); err != nil {
		return err
	}

	if _, err := parseDuration(s.Min); err != nil {
		return err
	}

	if s.Severity != "" {
		if _, err := checks.ParseSeverity(s.Severity); err != nil {
			return err
		}
	}

	return nil
}

func (s OffsetSettings) getSeverity(fallback checks.Severity) checks.Severity {
	if s.Severity != "" {
		sev, _ := checks.ParseSeverity(s.Severity)
		return sev
	}
	return fallback
}

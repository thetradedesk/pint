package config

import (
	"errors"
	"regexp"
)

type OffsetSettings struct {
	Prefix string `hcl:"regex" json:"regex,omitempty"`
	Min    string `hcl:"min" json:"min,omitempty"`
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

	return nil
}

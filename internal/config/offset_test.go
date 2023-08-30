package config

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOffsetSettings(t *testing.T) {
	type testCaseT struct {
		conf OffsetSettings
		err  error
	}

	testCases := []testCaseT{
		{
			conf: OffsetSettings{},
			err:  errors.New("offset prefix cannot be empty"),
		},
		{
			conf: OffsetSettings{
				Prefix: "aws_.*",
			},
			err: errors.New("minimum offset cannot be empty"),
		},
		{
			conf: OffsetSettings{
				Prefix: ".++",
				Min:    "10m",
			},
			err: errors.New("error parsing regexp: invalid nested repetition operator: `++`"),
		},
		{
			conf: OffsetSettings{
				Prefix: "aws_.*",
				Min:    "10123141",
			},
			err: errors.New("not a valid duration string: \"10123141\""),
		},
		{
			conf: OffsetSettings{
				Prefix: "azure_.*",
				Min:    "-10m",
			},
			err: errors.New("not a valid duration string: \"-10m\""),
		},
		{
			conf: OffsetSettings{
				Prefix: "azure_.*",
				Min:    "3c",
			},
			err: errors.New("unknown unit \"c\" in duration \"3c\""),
		},
		{
			conf: OffsetSettings{
				Prefix:   "azure_.*",
				Min:      "10m",
				Severity: "crab",
			},
			err: errors.New("unknown severity: crab"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.conf), func(t *testing.T) {
			err := tc.conf.validate()
			if err == nil || tc.err == nil {
				require.Equal(t, err, tc.err)
			} else {
				require.EqualError(t, err, tc.err.Error())
			}
		})
	}
}

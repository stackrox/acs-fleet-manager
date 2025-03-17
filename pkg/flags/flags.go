// Package flags is a helper package for processing and interactive command line flags
package flags

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// MustGetDefinedString attempts to get a non-empty string flag from the provided flag set or panic
func MustGetDefinedString(flagName string, flags *pflag.FlagSet) string {
	flagVal := MustGetString(flagName, flags)
	if flagVal == "" {
		panic(undefinedValueMessage(flagName))
	}
	return flagVal
}

// MustGetString attempts to get a string flag from the provided flag set or panic
func MustGetString(flagName string, flags *pflag.FlagSet) string {
	flagVal, err := flags.GetString(flagName)
	if err != nil {
		panic(notFoundMessage(flagName, err))
	}
	return flagVal
}

// MarkFlagRequired marks the given flag as required, panics if command has no flag with flagName
func MarkFlagRequired(flagName string, cmd *cobra.Command) {
	if err := cmd.MarkFlagRequired(flagName); err != nil {
		panic(err)
	}
}

func undefinedValueMessage(flagName string) string {
	return fmt.Sprintf("flag %s has undefined value", flagName)
}

func notFoundMessage(flagName string, err error) string {
	return fmt.Sprintf("could not get flag %s from flag set: %s", flagName, err.Error())
}

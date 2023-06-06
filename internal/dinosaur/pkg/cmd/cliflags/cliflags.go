// Package cliflags defines util methods for flags used in acsfleetctl
package cliflags

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// MarkFlagRequired marks the given flag as required, panics if command has no flag with flagName
func MarkFlagRequired(flagName string, cmd *cobra.Command) {
	if err := cmd.MarkFlagRequired(flagName); err != nil {
		panic(err)
	}
}

// MustGetDefinedString attempts to get a non-empty string flag from the provided command or panic
func MustGetDefinedString(flagName string, cmd *cobra.Command) string {
	flagVal := MustGetString(flagName, cmd)
	if flagVal == "" {
		panic(undefinedValueMessage(flagName))
	}
	return flagVal
}

// MustGetString attempts to get a string flag from the provided command or panic
func MustGetString(flagName string, cmd *cobra.Command) string {
	flagVal, err := cmd.Flags().GetString(flagName)
	if err != nil {
		panic(notFoundMessage(flagName, err))
	}
	return flagVal
}

// MustGetBool attempts to get a boolean flag from the provided command or panic
func MustGetBool(flagName string, cmd *cobra.Command) bool {
	flagVal, err := cmd.Flags().GetBool(flagName)
	if err != nil {
		glog.Fatalf(notFoundMessage(flagName, err))
	}
	return flagVal
}

func undefinedValueMessage(flagName string) string {
	return fmt.Sprintf("flag %s has undefined value", flagName)
}

func notFoundMessage(flagName string, err error) string {
	return fmt.Sprintf("could not get flag %s from flag set: %s", flagName, err.Error())
}

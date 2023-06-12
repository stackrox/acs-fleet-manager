// main package for acsfleetctl CLI
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/admin"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/centrals"
)

func main() {
	rootCmd := &cobra.Command{
		Use:  "acsfleetctl",
		Long: "acsfleetctl is a CLI used to interact with the ACSCS fleet-manager API",
	}
	rootCmd.PersistentFlags().Bool("debug", false, "use debug output")

	// This is used to recover from panics during initialization and command execution
	// use the --debug flag to print stacktrace otherwise only the panic msg will be printed
	defer recoverFromCLIPanic(rootCmd)

	setupSubCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

}

func recoverFromCLIPanic(rootCmd *cobra.Command) {
	if r := recover(); r != nil {
		fmt.Println(r)

		dbg, err := rootCmd.Flags().GetBool("debug")
		if err != nil {
			fmt.Println(err)
		}

		if dbg {
			fmt.Println(string(debug.Stack()))
		}

		os.Exit(1)
	}
}

func setupSubCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(centrals.NewCentralsCommand())
	rootCmd.AddCommand(admin.NewAdminCommand())
}

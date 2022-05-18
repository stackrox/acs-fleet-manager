package errors

import (
	"encoding/json"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"os"
	"sort"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	svcErr "github.com/stackrox/acs-fleet-manager/pkg/errors"
)

const (
	FlagsSaveToFile = "save-to-file"
)

// NewListCommand creates a new command for listing the errors which can be returned by the service.
func NewListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the errors which can be returned by the service",
		Long:  "List the errors which can be returned by the service",
		Run:   runList,
	}

	cmd.Flags().String(FlagsSaveToFile, "", "File path to save the list of errors in JSON format to (i.e. 'errors.json')")

	return cmd
}

func runList(cmd *cobra.Command, _ []string) {
	filePath := flags.MustGetString(FlagsSaveToFile, cmd.Flags())

	var svcErrors []public.Error
	errors := svcErr.Errors()

	// Sort errors by code
	sort.SliceStable(errors, func(i, j int) bool {
		return errors[i].Code < errors[j].Code
	})

	// add code prefix to service error code
	for _, err := range errors {
		svcErrors = append(svcErrors, handlers.PresentError(&err, ""))
	}

	svcErrorsJson, err := json.MarshalIndent(svcErrors, "", "\t")
	if err != nil {
		glog.Fatalf("failed to unmarshal struct")
	}

	// Write to stdout if filepath is not defined, otherwise save to the specified file
	if filePath == "" {
		glog.Infoln(string(svcErrorsJson))
	} else {
		file, err := os.Create(filePath)
		if err != nil {
			glog.Fatalf("failed to create file: %v", err)
		}
		defer file.Close()

		if _, err = file.WriteString(string(svcErrorsJson)); err != nil {
			glog.Fatalf("failed to write to file: %v", err)
		}
		glog.Infof("Service errors saved to %s", file.Name())
	}
}

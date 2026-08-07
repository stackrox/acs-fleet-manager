package centrals

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"
	"time"

	"os"

	"github.com/antihax/optional"
	"github.com/golang/glog"

	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	adminAPI "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/fleetmanagerclient"
)

var excludedStatuses = []string{
	constants.CentralRequestStatusFailed.String(),
}

// NewAdminReportCommand creates the admin report subcommand.
func NewAdminReportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Print ACS instance report in Slack mrkdwn format",
		Long:  "Print ACS instance report in Slack mrkdwn format.",
		Run: func(cmd *cobra.Command, args []string) {
			client := fleetmanagerclient.ClientFromContext(cmd.Context())
			if err := runReport(cmd.Context(), client.AdminAPI(), os.Stdout); err != nil {
				glog.Errorf("instance report failed: %v", err)
				os.Exit(1)
			}
		},
	}
}

// reportAPI is the subset of the fleet manager admin API needed for the report.
type reportAPI interface {
	GetCentrals(ctx context.Context, opts *adminAPI.GetCentralsOpts) (adminAPI.CentralList, *http.Response, error)
}

func fetchAllCentrals(ctx context.Context, api reportAPI, search string) ([]adminAPI.Central, error) {
	const pageSize = "100"
	var all []adminAPI.Central

	for page := 1; ; page++ {
		opts := &adminAPI.GetCentralsOpts{
			Page:    optional.NewString(fmt.Sprintf("%d", page)),
			Size:    optional.NewString(pageSize),
			OrderBy: optional.NewString("created_at desc"),
		}
		if search != "" {
			opts.Search = optional.NewString(search)
		}

		list, _, err := api.GetCentrals(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("fetching centrals (page %d): %w", page, err)
		}

		all = append(all, list.Items...)

		if int32(len(all)) >= list.Total {
			break
		}
	}
	return all, nil
}

func buildStatusFilter() string {
	var parts []string
	for _, s := range excludedStatuses {
		parts = append(parts, fmt.Sprintf("status <> %s", s))
	}
	return strings.Join(parts, " and ")
}

func runReport(ctx context.Context, api reportAPI, w io.Writer) error {
	statusFilter := buildStatusFilter()

	allCentrals, err := fetchAllCentrals(ctx, api, statusFilter)
	if err != nil {
		return fmt.Errorf("fetching centrals: %w", err)
	}

	sevenDaysAgo := time.Now().UTC().Add(-7 * 24 * time.Hour)

	var naInstances, euInstances, evalInstances, expiredInstances []adminAPI.Central

	for i := range allCentrals {
		c := &allCentrals[i]

		if strings.HasPrefix(c.Name, "probe-") {
			continue
		}

		if strings.HasPrefix(c.Region, "us-") {
			if !c.CreatedAt.Before(sevenDaysAgo) {
				naInstances = append(naInstances, *c)
			}
		}

		if strings.HasPrefix(c.Region, "eu-") {
			if !c.CreatedAt.Before(sevenDaysAgo) {
				euInstances = append(euInstances, *c)
			}
		}

		if c.InstanceType == "eval" {
			evalInstances = append(evalInstances, *c)
		}

		if c.ExpiredAt != nil {
			expiredInstances = append(expiredInstances, *c)
		}
	}

	allFailedCentrals, err := fetchAllCentrals(ctx, api, "status = failed")
	if err != nil {
		return fmt.Errorf("fetching failed centrals: %w", err)
	}
	var failedCentrals []adminAPI.Central
	for i := range allFailedCentrals {
		if !strings.HasPrefix(allFailedCentrals[i].Name, "probe-") {
			failedCentrals = append(failedCentrals, allFailedCentrals[i])
		}
	}

	today := time.Now().UTC().Format("2006-01-02")
	fmt.Fprintf(w, "*Instance Report* (%s)\n", today)
	fmt.Fprintln(w, "I live in <https://github.com/stackrox/acs-fleet-manager|stackrox/acs-fleet-manager>, you can get your own copy by running `./fleet-manager admin central report` there.")

	regionHeaders := []string{"ID", "Name", "Org", "Owner", "Created", "Type/Quota", "Status"}
	writeSection(w, "All *new* instances in North America in the past *7 days*:", naInstances, regionHeaders, regionRow)
	writeSection(w, "All *new* instances in Europe in the past *7 days*:", euInstances, regionHeaders, regionRow)

	evalHeaders := []string{"ID", "Name", "Org", "Owner", "Created", "Region", "Type/Quota", "Status"}
	writeSection(w, "All *eval* instances (we want none):", evalInstances, evalHeaders, fullRow)

	expiredHeaders := []string{"ID", "Name", "Org", "Owner", "Expires", "Region", "Status"}
	writeSection(w, "Instances that have *expiration date set*, they are usually removed 2 weeks after:", expiredInstances, expiredHeaders, expiredRow)

	failedHeaders := []string{"ID", "Name", "Org", "Owner", "Region", "Type/Quota", "Failed Reason"}
	writeSection(w, "All *failed* instances:", failedCentrals, failedHeaders, failedRow)

	return nil
}

type rowFunc func(c *adminAPI.Central) []string

func writeSection(w io.Writer, title string, instances []adminAPI.Central, headers []string, rowFn rowFunc) {
	fmt.Fprintf(w, "\n%s [%d]\n", title, len(instances))
	if len(instances) == 0 {
		fmt.Fprintln(w, "(none)")
		return
	}
	fmt.Fprintln(w, "```")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	writeRow(tw, headers)
	for i := range instances {
		writeRow(tw, rowFn(&instances[i]))
	}
	tw.Flush()
	fmt.Fprintln(w, "```")
}

func writeRow(w io.Writer, values []string) {
	for i, v := range values {
		if i > 0 {
			fmt.Fprint(w, "\t| ")
		}
		fmt.Fprint(w, v)
	}
	fmt.Fprintln(w)
}

func orgColumn(c *adminAPI.Central) string {
	name := c.OrganisationName
	if name == "" {
		name = "-"
	}
	if c.OrganisationId == "" {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, c.OrganisationId)
}

func typeQuota(c *adminAPI.Central) string {
	if c.QuotaType == "" {
		return c.InstanceType
	}
	return c.InstanceType + "/" + c.QuotaType
}

func regionRow(c *adminAPI.Central) []string {
	return []string{c.Id, c.Name, orgColumn(c), c.Owner, c.CreatedAt.Format("2006-01-02"),
		typeQuota(c), c.Status}
}

func fullRow(c *adminAPI.Central) []string {
	return []string{c.Id, c.Name, orgColumn(c), c.Owner, c.CreatedAt.Format("2006-01-02"), c.Region,
		typeQuota(c), c.Status}
}

func expiredRow(c *adminAPI.Central) []string {
	expires := "-"
	if c.ExpiredAt != nil {
		expires = c.ExpiredAt.Format("2006-01-02 15:04")
	}
	return []string{c.Id, c.Name, orgColumn(c), c.Owner, expires, c.Region, c.Status}
}

func failedRow(c *adminAPI.Central) []string {
	reason := c.FailedReason
	if reason == "" {
		reason = "-"
	}
	return []string{c.Id, c.Name, orgColumn(c), c.Owner, c.Region,
		typeQuota(c), reason}
}

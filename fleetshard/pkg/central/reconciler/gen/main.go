// This program generates a list of GVKs used by the tenant-resources helm chart for enabling garbage collection.
//
// To enable garbage collection without this list, we would need to...
// - Manually maintain a list of GVKs present in the chart, which would be error-prone.
// - Or have the reconciler list all objects for all possible GVKs, which would be very expensive.
// - Switch to the native helm client or the helm operator, which would require a lot of changes.
//
// This program automatically generates that list.
//
// Process:
// - It extract the GVKs that are present in the tenant-resources chart manifests
// - It extract the GVKs that are already declared in the generated file
// - It combines the two lists
// - It writes the combined list to the generated file

package main

import (
	"fmt"
	"github.com/pkg/errors"
	"io/fs"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

var (
	reconcilerDir           = path.Join(path.Dir(getCurrentFile()), "..")
	outFile                 = fmt.Sprintf("%s/zzz_managed_resources.go", reconcilerDir)
	tenantResourcesChartDir = path.Join(path.Dir(getCurrentFile()), "../../charts/data/tenant-resources/templates")
	gvkRegex                = regexp.MustCompile(`schema.GroupVersionKind{Group: "(.*)", Version: "(.*)", Kind: "(.*)"},`)
)

const (
	apiVersionPrefix = "apiVersion: "
	kindPrefix       = "kind: "
)

func main() {
	if err := generate(); err != nil {
		panic(err)
	}
}

type resourceMap map[schema.GroupVersionKind]bool

func generate() error {

	// Holds a map of GVKs that will be in the output file
	// The value `bool` indicates that the resource is present in the tenant-resources helm chart
	// If false, this means that the resource is only present in the generated file (for GVKs removed from the chart)
	// But the GVK is still needed for garbage collection.
	seen := resourceMap{}

	// Finding GVKs used in the tenant-resources chart
	if err := findGVKsInChart(tenantResourcesChartDir, seen); err != nil {
		return err
	}

	// Finding GVKs already declared in the generated file
	if err := findGVKsInGeneratedFile(seen); err != nil {
		return err
	}

	// Re-generating the file
	if err := generateGVKsList(seen); err != nil {
		return err
	}

	return nil
}

func getCurrentFile() string {
	_, file, _, _ := runtime.Caller(0)
	return file
}

func generateGVKsList(seen resourceMap) error {
	// Making sure resources are ordered (for deterministic output)
	sorted := sortResourceKeys(seen)

	builder := strings.Builder{}
	builder.WriteString("// Code generated by fleetshard/pkg/central/reconciler/gen/certmonitor.go. DO NOT EDIT.\n")
	builder.WriteString("package reconciler\n\n")
	builder.WriteString("import (\n")
	builder.WriteString("\t\"k8s.io/apimachinery/pkg/runtime/schema\"\n")
	builder.WriteString(")\n\n")

	builder.WriteString("// tenantChartResourceGVKs is a list of GroupVersionKind that...\n")
	builder.WriteString("// - are present in the tenant-resources helm chart\n")
	builder.WriteString("// - were present in a previous version of the chart. A comment will indicate that manual removal from the list is required.\n")

	builder.WriteString("var tenantChartResourceGVKs = []schema.GroupVersionKind{\n")
	for _, k := range sorted {
		builder.WriteString(fmt.Sprintf("\tschema.GroupVersionKind{Group: %q, Version: %q, Kind: %q},", k.Group, k.Version, k.Kind))
		stillInChart := seen[k]
		if !stillInChart {
			builder.WriteString(" // This resource was present in a previous version of the chart. Manual removal is required.")
		}
		builder.WriteString("\n")
	}
	builder.WriteString("}\n")

	genFile, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer genFile.Close()

	genFile.WriteString(builder.String())
	return nil
}

func sortResourceKeys(seen resourceMap) []schema.GroupVersionKind {
	sorted := make([]schema.GroupVersionKind, 0, len(seen))
	for k := range seen {
		sorted = append(sorted, k)
	}
	sort.Slice(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		return left.String() < right.String()
	})
	return sorted
}

func findGVKsInGeneratedFile(seen resourceMap) error {
	if _, err := os.Stat(outFile); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	file, err := os.Open(outFile)
	if err != nil {
		return err
	}
	defer file.Close()

	fileBytes, err := os.ReadFile(outFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(fileBytes), "\n")

	for _, line := range lines {
		matches := gvkRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}
		gvk := schema.GroupVersionKind{Group: matches[1], Version: matches[2], Kind: matches[3]}
		isAlreadyPresent := seen[gvk]
		seen[gvk] = isAlreadyPresent
	}
	return nil
}

func findGVKsInChart(chartDir string, seen resourceMap) error {
	if err := filepath.WalkDir(chartDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		fileBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := findGVKsInFile(fileBytes, seen); err != nil {
			return fmt.Errorf("failed to parse file %q: %w", path, err)
		}

		return nil
	}); err != nil {
		return err
	}
	return nil
}

func splitFileIntoResources(fileBytes []byte) []string {
	fileStr := string(fileBytes)
	return strings.Split(fileStr, "---")
}

func findGVKsInFile(fileBytes []byte, seen resourceMap) error {
	resources := splitFileIntoResources(fileBytes)
	for _, resource := range resources {
		gvk, ok, err := findResourceGVK(resource)
		if err != nil {
			return fmt.Errorf("failed to parse resource: %w", err)
		}
		if !ok {
			return errors.New("resource GVK could not be parsed")
		}
		seen[gvk] = true
	}
	return nil
}

func findResourceGVK(resource string) (schema.GroupVersionKind, bool, error) {
	lines := strings.Split(resource, "\n")
	apiVersion := ""
	kind := ""
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], apiVersionPrefix) {
			apiVersion = strings.TrimSpace(strings.TrimPrefix(lines[i], apiVersionPrefix))
			continue
		}
		if strings.HasPrefix(lines[i], kindPrefix) {
			kind = strings.TrimSpace(strings.TrimPrefix(lines[i], kindPrefix))
			continue
		}
	}
	if len(apiVersion) == 0 || len(kind) == 0 {
		return schema.GroupVersionKind{}, false, nil
	}

	groupVersion, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, false, err
	}

	return groupVersion.WithKind(kind), true, nil
}

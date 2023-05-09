// Package charts ...
package charts

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/golang/glog"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var (
	// The templates/* entry is necessary because files starting with an underscore are only embedded when matched
	// via *, not when recursively traversing a directory. Once we switch to go1.18, we can change the embed spec
	// to all:data.
	//go:embed data data/tenant-resources/templates/*
	data embed.FS
)

// TraverseChart combines all chart files into memory from given file system
func TraverseChart(fsys fs.FS, chartPath string) ([]*loader.BufferedFile, error) {
	chartPath = strings.TrimRight(chartPath, "/")
	var chartFiles []*loader.BufferedFile
	err := fs.WalkDir(fsys, chartPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		bytes, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}
		chartFiles = append(chartFiles, &loader.BufferedFile{
			Name: path[len(chartPath)+1:], // strip "<path>/"
			Data: bytes,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("loading chart from %q: %w", chartPath, err)
	}
	return chartFiles, nil
}

func downloadTemplates(urls []string) ([]*loader.BufferedFile, error) {
	var chartFiles []*loader.BufferedFile
	for _, url := range urls {
		buffered, err := downloadTemplate(url)
		if err != nil {
			return nil, fmt.Errorf("failed downloading template from %s: %w", url, err)
		}
		chartFiles = append(chartFiles, buffered)
	}
	return chartFiles, nil
}

func downloadTemplate(url string) (*loader.BufferedFile, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed Get for %s: %w", url, err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read bytes: %w", err)
	}

	// parse filename from the URL
	filename := url[strings.LastIndex(url, "/")+1:]
	name := path.Join("templates", filename)

	bufferedFile := &loader.BufferedFile{
		Name: name,
		Data: bytes,
	}

	return bufferedFile, nil
}

// GetChart loads a chart from the data directory. The name should be the name of the containing directory.
// Optional: pass list of URLs to download additional template files for the chart.
func GetChart(name string, urls []string) (*chart.Chart, error) {
	chartFiles, err := TraverseChart(data, path.Join("data", name))
	if err != nil {
		return nil, fmt.Errorf("failed getting chart files for %q: %w", name, err)
	}
	if len(urls) > 0 {
		downloadedFiles, err := downloadTemplates(urls)
		if err != nil {
			return nil, fmt.Errorf("failed downloading chart files %q: %w", name, err)
		}
		chartFiles = append(chartFiles, downloadedFiles...)
	}
	loadedChart, err := loader.LoadFiles(chartFiles)
	if err != nil {
		return nil, fmt.Errorf("failed loading chart %q: %w", name, err)
	}
	return loadedChart, nil
}

// MustGetChart loads a chart from the data directory. Unlike GetChart, it panics if an error is encountered.
func MustGetChart(name string, urls []string) *chart.Chart {
	chrt, err := GetChart(name, urls)
	if err != nil {
		panic(err)
	}
	return chrt
}

// InstallOrUpdateChart installs a new object from helm chart or update an existing object with the same Name, Namespace and Kind
func InstallOrUpdateChart(ctx context.Context, obj *unstructured.Unstructured, client ctrlClient.Client) error {
	key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	var out unstructured.Unstructured
	out.SetGroupVersionKind(obj.GroupVersionKind())
	err := client.Get(ctx, key, &out)
	if err == nil {
		glog.V(10).Infof("Updating object %s/%s", obj.GetNamespace(), obj.GetName())
		obj.SetResourceVersion(out.GetResourceVersion())
		err := client.Update(ctx, obj)
		if err != nil {
			return fmt.Errorf("failed to update object %s/%s of type %s %w", key.Namespace, key.Name, obj.GetKind(), err)
		}
	} else {
		if !apiErrors.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve object %s/%s of type %s %w", key.Namespace, key.Name, obj.GetKind(), err)
		}
		err = client.Create(ctx, obj)
		glog.Infof("Creating object %s/%s", obj.GetNamespace(), obj.GetName())
		if err != nil && !apiErrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create object %s/%s of type %s: %w", key.Namespace, key.Name, obj.GetKind(), err)
		}
	}
	return nil
}

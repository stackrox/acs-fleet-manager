// Package charts ...
package charts

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
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

// LoadChart loads a chart from the given path on the given file system.
func LoadChart(fsys fs.FS, chartPath string) (*chart.Chart, error) {
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

	chrt, err := loader.LoadFiles(chartFiles)
	if err != nil {
		return nil, fmt.Errorf("loading chart from %s: %w", chartPath, err)
	}
	return chrt, nil
}

// GetChart loads a chart from the data directory. The name should be the name of the containing directory.
func GetChart(name string) (*chart.Chart, error) {
	return LoadChart(data, path.Join("data", name))
}

// MustGetChart loads a chart from the data directory. Unlike GetChart, it panics if an error is encountered.
func MustGetChart(name string) *chart.Chart {
	chrt, err := GetChart(name)
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

// DownloadTemplate downloads a file from the URL to the templates folder of specified chart
func DownloadTemplate(url string, chartName string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed Get for %s: %w", url, err)
	}
	defer resp.Body.Close()

	// generate the filename from the URL
	filename := url[strings.LastIndex(url, "/")+1:]
	filepath := path.Join("data", chartName, "templates", filename)

	// Make sure that file will be created in charts/data/<chart>/templates folder
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get working directory %w", err)
	}
	parent := path.Dir(wd)
	err = os.Chdir(path.Join(parent, "charts"))
	if err != nil {
		return fmt.Errorf("cannot change working directory %w", err)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("cannot create file %s: %w", filepath, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("cannot copy to file %s: %w", filepath, err)
	}

	return nil
}

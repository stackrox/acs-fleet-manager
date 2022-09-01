package charts

import (
	"embed"
	"fmt"
	"io/fs"
	"path"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var (
	//go:embed all:data
	data embed.FS
)

// GetChart loads a chart from the data directory. The name should be the name of the containing directory.
func GetChart(name string) (*chart.Chart, error) {
	var chartFiles []*loader.BufferedFile
	dirPrefix := path.Join("data", name)
	err := fs.WalkDir(data, dirPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		bytes, err := fs.ReadFile(data, path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}
		chartFiles = append(chartFiles, &loader.BufferedFile{
			Name: path[len(dirPrefix)+1:], // strip "<dirPrefix>/"
			Data: bytes,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("loading chart %q: %w", name, err)
	}

	chrt, err := loader.LoadFiles(chartFiles)
	if err != nil {
		return nil, fmt.Errorf("loading chart %s: %w", name, err)
	}
	return chrt, nil
}

// MustGetChart loads a chart from the data directory. Unlike GetChart, it panics if an error is encountered.
func MustGetChart(name string) *chart.Chart {
	chrt, err := GetChart(name)
	if err != nil {
		panic(err)
	}
	return chrt
}

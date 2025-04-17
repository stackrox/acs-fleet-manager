package gitops

import (
	// embed needed for embedding the default central template
	_ "embed"
	"os"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// Reader reads a Config from a source.
type Reader interface {
	Read() (Config, error)
}

// fileReader is a Reader that reads a Config from a file.
type fileReader struct {
	path string
}

// NewFileReader returns a new fileReader.
func NewFileReader(path string) Reader {
	return &fileReader{path: path}
}

// Read implements Reader.Read
func (r *fileReader) Read() (Config, error) {
	fileBytes, err := os.ReadFile(r.path)
	if err != nil {
		return Config{}, errors.Wrap(err, "failed to read GitOps configuration file")
	}
	var config Config
	if err := yaml.Unmarshal(fileBytes, &config); err != nil {
		return Config{}, errors.Wrap(err, "failed to unmarshal GitOps configuration")
	}
	return config, nil
}

// staticReader is a Reader that returns a static Config.
type staticReader struct {
	config Config
}

// NewStaticReader returns a new staticReader.
// Useful for testing.
func NewStaticReader(config Config) Reader {
	return &staticReader{config: config}
}

// Read implements Reader.Read
func (r *staticReader) Read() (Config, error) {
	return r.config, nil
}

// emptyReader is a Reader that returns an empty Config.
type emptyReader struct{}

// NewEmptyReader returns a new emptyReader.
// Useful for testing.
func NewEmptyReader() Reader {
	return &emptyReader{}
}

// Read implements Reader.Read
func (r *emptyReader) Read() (Config, error) {
	return Config{}, nil
}

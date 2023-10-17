package gitops

import (
	"context"
	"os"
	"sync"

	// embed needed for embedding the default central template
	_ "embed"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

type configMapReader struct {
	name      string
	key       string
	client    kubernetes.Interface
	lock      sync.RWMutex
	val       Config
	namespace string
}

// NewConfigMapReader returns a new configMapReader.
func NewConfigMapReader(ctx context.Context, namespace, name, key string, client kubernetes.Interface) Reader {
	r := &configMapReader{
		namespace: namespace,
		name:      name,
		key:       key,
		client:    client,
	}
	watcher := k8scfgwatch.NewConfigMapWatcher(r.client, func(configMap *corev1.ConfigMap) {
		glog.Infof("received new GitOps configuration")
		var config Config
		if err := yaml.Unmarshal([]byte(configMap.Data[r.key]), &config); err != nil {
			glog.Errorf("failed to unmarshal GitOps configuration: %v", err)
			return
		}
		r.lock.Lock()
		defer r.lock.Unlock()
		r.val = config
	})
	watcher.Watch(ctx, namespace, name)
	return r
}

func (r *configMapReader) Read() (Config, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.val, nil
}

const (
	defaultConfigMapName = "fleet-manager-gitops-config"
	configMapNameEnvVar  = "RHACS_GITOPS_CONFIGMAP_NAME"
	configMapKey         = "config.yaml"
)

// NewReader returns a new gitops Reader. Will
// return an empty reader if GitOps is not enabled.
// Otherwise returns a ConfigMap reader.
func NewReader() Reader {
	if !features.GitOpsCentrals.Enabled() {
		return NewEmptyReader()
	}

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("failed to get in-cluster config: %v", err)
		return NewEmptyReader()
	}

	k8sInterface, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		glog.Errorf("failed to create kubernetes client: %v", err)
		return NewEmptyReader()
	}

	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		glog.Errorf("failed to get POD_NAMESPACE env var")
		return NewEmptyReader()
	}

	cmName := os.Getenv(configMapNameEnvVar)
	if cmName == "" {
		cmName = defaultConfigMapName
	}

	return NewConfigMapReader(context.Background(), ns, cmName, configMapKey, k8sInterface)
}

package presenters

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestShouldNotRenderTwiceForSameParams(t *testing.T) {
	var gitopsConfig = gitops.Config{}
	var params = gitops.CentralParams{}
	var renderCount = 0
	r := newCachedCentralRenderer()
	r.renderFn = func(params gitops.CentralParams, config gitops.Config) (v1alpha1.Central, error) {
		renderCount++
		return v1alpha1.Central{}, nil
	}

	assert.Equal(t, 0, renderCount)

	// first call should render once
	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 1, renderCount)

	// second call should not render again
	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 1, renderCount)

	// third call with different params should render again
	params = gitops.CentralParams{ID: "foo"}
	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 2, renderCount)

	// fourth call with same params should not render again
	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 2, renderCount)

	// fifth call with different params should render again
	gitopsConfig = gitops.Config{Centrals: gitops.CentralsConfig{Overrides: []gitops.CentralOverride{{InstanceIDs: []string{"foo"}}}}}
	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 3, renderCount)

	// sixth call with same params should not render again
	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 3, renderCount)
}

func TestShouldNotCacheOnError(t *testing.T) {
	var gitopsConfig = gitops.Config{}
	var params = gitops.CentralParams{}
	var renderCount = 0
	var shouldThrow = false

	r := newCachedCentralRenderer()
	r.renderFn = func(params gitops.CentralParams, config gitops.Config) (v1alpha1.Central, error) {
		renderCount++
		if shouldThrow {
			return v1alpha1.Central{}, assert.AnError
		}
		return v1alpha1.Central{}, nil
	}

	assert.Equal(t, 0, renderCount)

	shouldThrow = true
	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 1, renderCount)

	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 2, renderCount)

	shouldThrow = false
	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 3, renderCount)

	r.getCentralYaml(gitopsConfig, params)
	assert.Equal(t, 3, renderCount)
}

func TestKeyedMutexShouldLockByKey(t *testing.T) {
	m := newKeyedMutex()

	// locking "foo"
	m.lock("foo")

	// foo is locked
	assert.True(t, m.isLocked("foo"))
	assert.False(t, m.isLocked("bar"))

	// unlock foo
	m.unlock("foo")
	assert.False(t, m.isLocked("foo"))

}

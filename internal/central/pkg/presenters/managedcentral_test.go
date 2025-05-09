package presenters

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops"
	"github.com/stretchr/testify/assert"
)

func TestShouldNotRenderTwiceForSameParams(t *testing.T) {
	var gitopsConfig = gitops.Config{}
	var params = gitops.CentralParams{}
	var renderCount = 0
	r := newCachedCentralRenderer()
	r.renderValuesFn = func(params gitops.CentralParams, config gitops.Config) (map[string]interface{}, error) {
		renderCount++
		return map[string]interface{}{}, nil
	}

	assert.Equal(t, 0, renderCount)

	// first call should render once
	r.render(gitopsConfig, params)
	assert.Equal(t, 1, renderCount)

	// second call should not render again
	r.render(gitopsConfig, params)
	assert.Equal(t, 1, renderCount)

	// third call with different params should render again
	params = gitops.CentralParams{ID: "foo"}
	r.render(gitopsConfig, params)
	assert.Equal(t, 2, renderCount)

	// fourth call with same params should not render again
	r.render(gitopsConfig, params)
	assert.Equal(t, 2, renderCount)

	// fifth call with different params should render again
	gitopsConfig = gitops.Config{TenantResources: gitops.TenantResourceConfig{Overrides: []gitops.TenantResourceOverride{{InstanceIDs: []string{"foo"}}}}}
	r.render(gitopsConfig, params)
	assert.Equal(t, 3, renderCount)

	// sixth call with same params should not render again
	r.render(gitopsConfig, params)
	assert.Equal(t, 3, renderCount)
}

func TestShouldNotCacheOnError(t *testing.T) {
	var gitopsConfig = gitops.Config{}
	var params = gitops.CentralParams{}
	var renderCount = 0
	var shouldThrow = false

	r := newCachedCentralRenderer()
	r.renderValuesFn = func(params gitops.CentralParams, config gitops.Config) (map[string]interface{}, error) {
		renderCount++
		if shouldThrow {
			return map[string]interface{}{}, assert.AnError
		}
		return map[string]interface{}{}, nil
	}

	assert.Equal(t, 0, renderCount)

	shouldThrow = true
	r.render(gitopsConfig, params)
	assert.Equal(t, 1, renderCount)

	r.render(gitopsConfig, params)
	assert.Equal(t, 2, renderCount)

	shouldThrow = false
	r.render(gitopsConfig, params)
	assert.Equal(t, 3, renderCount)

	r.render(gitopsConfig, params)
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

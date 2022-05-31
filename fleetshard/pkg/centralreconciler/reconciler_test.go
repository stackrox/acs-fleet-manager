package centralreconciler

import (
	"fmt"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
	"testing"
)

func TestTriggerOnlyOneReconcile(t *testing.T) {
	t.Skip("skipping test...")
	errors := make(chan ReconcilerResult, 25)
	reconciler := CentralReconciler{
		resultCh: errors,
		inputCh:  make(chan *private.ManagedCentral, 10),
		status:   pointer.Int32(0),
	}

	go reconciler.Start()

	for i := 0; i < 99; i++ {
		count := i
		go func() {
			reconciler.InputChannel() <- &private.ManagedCentral{Id: fmt.Sprint(count)}
			fmt.Println("send request")
		}()
	}

	counter := 0
	for err := range errors {
		counter++
		fmt.Println(err.Err)
		if counter == 99 {
			break
		}
	}

	assert.Equal(t, 99, counter)
}

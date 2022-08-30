package rxgo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNever(t *testing.T) {
	// NEVER[any]().SubscribeSync(func(a any) {}, func(err error) {}, func() {
	// 	log.Println("Completed")
	// })
}

func TestEmpty(t *testing.T) {

}

func TestThrownError(t *testing.T) {
	var v = fmt.Errorf("uncaught error")
	var isComplete bool
	ThrownError[string](func() error {
		return v
	}).SubscribeSync(
		noop[string],
		func(err error) {
			require.Equal(t, v, err)
		}, func() {
			isComplete = true
		})
	require.True(t, isComplete)
}

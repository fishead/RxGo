package rxgo

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestNever(t *testing.T) {
}

func TestEmpty(t *testing.T) {
	t.Run(`Empty with "any" type`, func(t *testing.T) {
		checkObservableResult(t, Empty[any](), nil, nil, true)
		checkObservableResults(t, Empty[any](), []any{}, nil, true)
	})

	t.Run(`Empty with "string" type`, func(t *testing.T) {
		checkObservableResult(t, Empty[string](), "", nil, true)
		checkObservableResults(t, Empty[string](), []string{}, nil, true)
	})

	t.Run(`Empty with "uint" type`, func(t *testing.T) {
		checkObservableResult(t, Empty[uint](), uint(0), nil, true)
		checkObservableResults(t, Empty[uint](), []uint{}, nil, true)
	})
}

func TestThrow(t *testing.T) {
	var v = fmt.Errorf("uncaught error")
	checkObservableResults(t, Throw[string](func() error {
		return v
	}), []string{}, v, false)
}

func TestDefer(t *testing.T) {
	t.Run("Defer with nil", func(t *testing.T) {
		checkObservableResult(t, Defer(func() Observable[string] {
			return nil
		}), "", nil, true)
	})

	t.Run("Defer with Throw", func(t *testing.T) {
		var err = fmt.Errorf("throw")
		checkObservableResult(t, Defer(func() Observable[string] {
			return Throw[string](func() error {
				return err
			})
		}), "", err, false)
	})

	t.Run("Defer with alphaberts", func(t *testing.T) {
		values := []string{"a", "b", "c"}
		checkObservableResults(t, Defer(func() Observable[string] {
			return newObservable(func(subscriber Subscriber[string]) {
				for _, v := range values {
					subscriber.Send() <- Next(v)
				}
				subscriber.Send() <- Complete[string]()
			})
		}), values, nil, true)
	})
}

func TestRange(t *testing.T) {
	t.Run("Range from 1 to 10", func(t *testing.T) {
		checkObservableResults(t, Range[uint](1, 10), []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil, true)
	})
	t.Run("Range from 0 to 3", func(t *testing.T) {
		checkObservableResults(t, Range[uint](0, 3), []uint{0, 1, 2}, nil, true)
	})
}

func TestInterval(t *testing.T) {
	checkObservableResults(t, Pipe1(
		Interval(time.Millisecond),
		Take[uint](10),
	), []uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, nil, true)
}

func TestScheduled(t *testing.T) {
	t.Run("Scheduled with alphaberts", func(t *testing.T) {
		checkObservableResults(t, Scheduled("a", "q", "z"), []string{"a", "q", "z"}, nil, true)
	})

	t.Run("Scheduled with float32", func(t *testing.T) {
		checkObservableResults(t, Scheduled[float32](18.24, 1.776, 88), []float32{18.24, 1.776, 88}, nil, true)
	})

	t.Run("Scheduled with error", func(t *testing.T) {
		var err = errors.New("something wrong")
		checkObservableResults(t, Scheduled[any]("a", 1, err, 88), []any{"a", 1}, err, false)
	})
}

func TestTimer(t *testing.T) {
	t.Run("Timer with zero initial value", func(t *testing.T) {
		checkObservableResult(t, Timer[uint](0), uint(0), nil, true)
	})

	t.Run("Timer with non-zero initial value", func(t *testing.T) {
		checkObservableResult(t, Timer[uint](time.Millisecond), uint(0), nil, true)
	})

	t.Run("Timer with initial and duration value", func(t *testing.T) {
		checkObservableResults(t, Pipe1(
			Timer[uint](0, time.Millisecond),
			Take[uint](3),
		), []uint{0, 1, 2}, nil, true)
	})
}

func TestIif(t *testing.T) {
	t.Run("Iif with Empty and Interval", func(t *testing.T) {
		flag := true
		iif := Iif(func() bool {
			return flag
		}, Empty[uint](), Pipe1(Interval(time.Millisecond), Take[uint](3)))
		checkObservableResult(t, iif, uint(0), nil, true)
		flag = false
		checkObservableResults(t, iif, []uint{0, 1, 2}, nil, true)
	})

	t.Run("Iif with Scheduled and Empty", func(t *testing.T) {
		iif := Iif(func() bool {
			return true
		}, Scheduled("a", "q", "%", "@"), Empty[string]())
		checkObservableResults(t, iif, []string{"a", "q", "%", "@"}, nil, true)
	})

	t.Run("Iif with error", func(t *testing.T) {
		var err = errors.New("throw")
		iif := Iif(func() bool {
			return true
		}, Scheduled[any]("a", err, "%", "@"), Empty[any]())
		checkObservableResults(t, iif, []any{"a"}, err, false)
	})
}

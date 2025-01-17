package rxgo

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRepeat(t *testing.T) {
	t.Run("Repeat with Empty", func(t *testing.T) {
		checkObservableResults(t, Pipe1(
			Empty[any](),
			Repeat[any, uint](3),
		), []any{}, nil, true)
	})

	t.Run("Repeat with config", func(t *testing.T) {
		checkObservableResults(t, Pipe1(
			Of2[any](12, 88),
			Repeat[any](RepeatConfig{
				Count: 5,
				Delay: time.Millisecond,
			}),
		), []any{12, 88, 12, 88, 12, 88, 12, 88, 12, 88}, nil, true)
	})

	t.Run("Repeat with error", func(t *testing.T) {
		var err = errors.New("throw")
		// Repeat with error will no repeat
		checkObservableResults(t, Pipe1(
			Throw[string](func() error {
				return err
			}),
			Repeat[string, uint](3),
		), []string{}, err, false)
	})

	t.Run("Repeat with error", func(t *testing.T) {
		var err = errors.New("throw at 3")
		// Repeat with error will no repeat
		checkObservableResults(t, Pipe2(
			Range[uint](1, 10),
			Map(func(v, _ uint) (uint, error) {
				if v >= 3 {
					return 0, err
				}
				return v, nil
			}),
			Repeat[uint, uint](3),
		), []uint{1, 2}, err, false)
	})

	t.Run("Repeat with alphaberts", func(t *testing.T) {
		checkObservableResults(t, Pipe1(
			Of2("a", "bb", "cc", "ff", "gg"),
			Repeat[string, uint](3),
		), []string{
			"a", "bb", "cc", "ff", "gg",
			"a", "bb", "cc", "ff", "gg",
			"a", "bb", "cc", "ff", "gg",
		}, nil, true)
	})
}

func TestDo(t *testing.T) {
	t.Run("Do with Range(1, 5)", func(t *testing.T) {
		result := make([]string, 0)
		checkObservableResults(t, Pipe1(
			Range[uint](1, 5),
			Do(NewObserver(func(v uint) {
				result = append(result, fmt.Sprintf("Number(%v)", v))
			}, nil, nil)),
		), []uint{1, 2, 3, 4, 5}, nil, true)
		require.ElementsMatch(t, []string{
			"Number(1)",
			"Number(2)",
			"Number(3)",
			"Number(4)",
			"Number(5)",
		}, result)
	})

	t.Run("Do with Error", func(t *testing.T) {
		var (
			err    = fmt.Errorf("An error")
			result = make([]string, 0)
		)
		checkObservableResults(t, Pipe1(
			Scheduled[any](1, err),
			Do(NewObserver(func(v any) {
				result = append(result, fmt.Sprintf("Number(%v)", v))
			}, nil, nil)),
		), []any{1}, err, false)
		require.ElementsMatch(t, []string{"Number(1)"}, result)
	})
}

func TestDelay(t *testing.T) {
	t.Run("Delay", func(t *testing.T) {
		checkObservableResults(t, Pipe1(
			Range[uint8](1, 5),
			Delay[uint8](time.Millisecond*50),
		), []uint8{1, 2, 3, 4, 5}, nil, true)
	})
}

func TestDelayWhen(t *testing.T) {
	t.Run("DelayWhen", func(t *testing.T) {})
}

func TestTimeout(t *testing.T) {
	t.Run("Timeout with Empty", func(t *testing.T) {
		checkObservableResult(t, Pipe1(
			Empty[any](),
			Timeout[any](time.Second),
		), nil, nil, true)
	})

	t.Run("Timeout with error", func(t *testing.T) {
		var err = errors.New("failed")
		checkObservableResult(t, Pipe1(
			Throw[any](func() error {
				return err
			}),
			Timeout[any](time.Millisecond),
		), nil, err, false)
	})

	t.Run("Timeout with timeout error", func(t *testing.T) {
		checkObservableResult(t, Pipe1(
			Interval(time.Millisecond*10),
			Timeout[uint](time.Millisecond),
		), uint(0), ErrTimeout, false)
	})

	t.Run("Timeout with Scheduled", func(t *testing.T) {
		checkObservableResult(t, Pipe1(
			Of2("a"),
			Timeout[string](time.Millisecond*100),
		), "a", nil, true)
	})
}

func TestToSlice(t *testing.T) {
	t.Run("ToSlice with Empty", func(t *testing.T) {
		checkObservableResult(t, Pipe1(Empty[any](), ToSlice[any]()), []any{}, nil, true)
	})

	t.Run("ToSlice with error", func(t *testing.T) {
		var err = errors.New("throw")
		checkObservableResult(t, Pipe1(Scheduled[any]("a", "z", err), ToSlice[any]()),
			nil, err, false)
	})

	t.Run("ToSlice with numbers", func(t *testing.T) {
		checkObservableResult(t, Pipe1(Range[uint](1, 5), ToSlice[uint]()), []uint{1, 2, 3, 4, 5}, nil, true)
	})

	t.Run("ToSlice with alphaberts", func(t *testing.T) {
		checkObservableResult(t, Pipe1(newObservable(func(subscriber Subscriber[string]) {
			for i := 1; i <= 5; i++ {
				Next(string(rune('A' - 1 + i))).Send(subscriber)
			}
			Complete[string]().Send(subscriber)
		}), ToSlice[string]()), []string{"A", "B", "C", "D", "E"}, nil, true)
	})
}

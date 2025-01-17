package rxgo

import (
	"sync"
	"time"

	"golang.org/x/exp/constraints"
)

type RepeatConfig struct {
	Count uint
	Delay time.Duration
}

type repeatConfig interface {
	constraints.Unsigned | RepeatConfig
}

// Returns an Observable that will resubscribe to the source stream when the source stream completes.
func Repeat[T any, C repeatConfig](config ...C) OperatorFunc[T, T] {
	var (
		maxRepeatCount = int64(-1)
		delay          = time.Duration(0)
	)

	if len(config) > 0 {
		switch v := any(config[0]).(type) {
		case RepeatConfig:
			if v.Count > 0 {
				maxRepeatCount = int64(v.Count)
			}
			if v.Delay > 0 {
				delay = v.Delay
			}
		case uint8:
			maxRepeatCount = int64(v)
		case uint16:
			maxRepeatCount = int64(v)
		case uint32:
			maxRepeatCount = int64(v)
		case uint64:
			maxRepeatCount = int64(v)
		case uint:
			maxRepeatCount = int64(v)
		}
	}

	return func(source Observable[T]) Observable[T] {
		return newObservable(func(subscriber Subscriber[T]) {
			var (
				wg          = new(sync.WaitGroup)
				repeatCount = int64(0)
				upStream    Subscriber[T]
				forEach     <-chan Notification[T]
			)

			setupStream := func(first bool) {
				wg.Add(1)
				if delay > 0 && !first {
					time.Sleep(delay)
				}
				upStream = source.SubscribeOn(wg.Done)
				forEach = upStream.ForEach()
			}

			setupStream(true)

		loop:
			for {
				select {
				case <-subscriber.Closed():
					upStream.Stop()
					break loop

				case item, ok := <-forEach:
					if !ok {
						break loop
					}

					if err := item.Err(); err != nil {
						Error[T](err).Send(subscriber)
						break loop
					}

					if item.Done() {
						repeatCount++
						if maxRepeatCount < 0 || repeatCount < maxRepeatCount {
							setupStream(false)
							continue
						}

						Complete[T]().Send(subscriber)
						break loop
					}

					item.Send(subscriber)
				}
			}

			wg.Wait()
		})
	}
}

// Used to perform side-effects for notifications from the source observable
func Do[T any](cb Observer[T]) OperatorFunc[T, T] {
	return func(source Observable[T]) Observable[T] {
		if cb == nil {
			cb = NewObserver[T](nil, nil, nil)
		}
		return createOperatorFunc(
			source,
			func(obs Observer[T], v T) {
				obs.Next(v)
				cb.Next(v)
			},
			func(obs Observer[T], err error) {
				obs.Error(err)
				cb.Error(err)
			},
			func(obs Observer[T]) {
				obs.Complete()
				cb.Complete()
			},
		)
	}
}

// Delays the emission of items from the source Observable by a given timeout.
func Delay[T any](duration time.Duration) OperatorFunc[T, T] {
	return func(source Observable[T]) Observable[T] {
		return createOperatorFunc(
			source,
			func(obs Observer[T], v T) {
				time.Sleep(duration)
				obs.Next(v)
			},
			func(obs Observer[T], err error) {
				obs.Error(err)
			},
			func(obs Observer[T]) {
				obs.Complete()
			},
		)
	}
}

// Delays the emission of items from the source Observable by a given time span determined by the emissions of another Observable.
func DelayWhen[T any, R any](delayDurationSelector ProjectionFunc[T, R]) OperatorFunc[T, T] {
	return func(source Observable[T]) Observable[T] {
		return newObservable(func(subscriber Subscriber[T]) {
			var (
				wg = new(sync.WaitGroup)
			)

			wg.Add(1)

			var (
				upStream = source.SubscribeOn(wg.Done)
				index    uint
			)

			observeStream := func(index uint, value T) {
				delayStream := delayDurationSelector(value, index).SubscribeOn(wg.Done)

			loop:
				for {
					select {
					case <-subscriber.Closed():
						delayStream.Stop()
						break loop

					case item, ok := <-delayStream.ForEach():
						if !ok {
							break loop
						}

						// If the "duration" Observable only emits the complete notification (without next), the value emitted by the source Observable will never get to the output Observable - it will be swallowed.
						if item.Done() {
							break loop
						}

						// If the "duration" Observable errors, the error will be propagated to the output Observable.
						if err := item.Err(); err != nil {
							Error[T](err).Send(subscriber)
							break loop
						}

						Next(value).Send(subscriber)
						delayStream.Stop()
						break loop
					}
				}
			}

		observe:
			for {
				select {
				case <-subscriber.Closed():
					upStream.Stop()
					break observe

				case item, ok := <-upStream.ForEach():
					if !ok {
						break observe
					}

					if item.Done() {
						break observe
					}

					if err := item.Err(); err != nil {
						Error[T](err).Send(subscriber)
						break observe
					}

					wg.Add(1)
					observeStream(index, item.Value())
					index++
				}
			}

			wg.Wait()
		})
	}
}

type TimeoutConfig[T any] struct {
	With func() Observable[T]
	Each time.Duration
}

type timeoutConfig[T any] interface {
	time.Duration | TimeoutConfig[T]
}

// Errors if Observable does not emit a value in given time span.
// FIXME:  DATA RACE and send on closed channel
func Timeout[T any, C timeoutConfig[T]](config C) OperatorFunc[T, T] {
	return func(source Observable[T]) Observable[T] {
		return newObservable(func(subscriber Subscriber[T]) {
			var (
				wg = new(sync.WaitGroup)
			)

			wg.Add(1)

			var (
				upStream = source.SubscribeOn(wg.Done)
				timer    *time.Timer
			)

			switch v := any(config).(type) {
			case time.Duration:
				timer = time.NewTimer(v)
			case TimeoutConfig[T]:
				panic("unimplemented")
			}

		loop:
			for {
				select {
				case <-subscriber.Closed():
					upStream.Stop()
					break loop

				case item, ok := <-upStream.ForEach():
					if !ok {
						break loop
					}

					// reset timeout
					timer.Stop()
					item.Send(subscriber)
					if item.Err() != nil || item.Done() {
						break loop
					}

				case <-timer.C:
					upStream.Stop()
					Error[T](ErrTimeout).Send(subscriber)
					break loop
				}
			}

			wg.Wait()
		})
	}
}

// Collects all source emissions and emits them as an array when the source completes.
func ToSlice[T any]() OperatorFunc[T, []T] {
	return func(source Observable[T]) Observable[[]T] {
		var (
			result = make([]T, 0)
		)
		return createOperatorFunc(
			source,
			func(obs Observer[[]T], v T) {
				result = append(result, v)
			},
			func(obs Observer[[]T], err error) {
				obs.Error(err)
			},
			func(obs Observer[[]T]) {
				obs.Next(result)
				obs.Complete()
			},
		)
	}
}

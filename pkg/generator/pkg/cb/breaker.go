package cb

import (
	"errors"
	"sync"
	"time"
)

type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	isSuccessful  func(err error) bool
	onStateChange func(name string, from State, to State)

	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
	mutex      sync.Mutex
}

func NewCircuitBreaker(name string, st Settings) *CircuitBreaker {

	cb := new(CircuitBreaker)
	cb.name = name
	cb.onStateChange = st.OnStateChange
	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}
	if st.Interval <= 0 {
		cb.interval = defaultInterval
	} else {
		cb.interval = st.Interval
	}
	if st.Timeout <= 0 {
		cb.timeout = defaultTimeout
	} else {
		cb.timeout = st.Timeout
	}
	if st.ReadyToTrip == nil {
		cb.readyToTrip = defaultReadyToTrip
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}
	if st.IsSuccessful == nil {
		cb.isSuccessful = defaultIsSuccessful
	} else {
		cb.isSuccessful = st.IsSuccessful
	}
	cb.toNewGeneration(time.Now())
	return cb
}

func (cb *CircuitBreaker) Name() string {
	return cb.name
}

func (cb *CircuitBreaker) State() State {

	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

func (cb *CircuitBreaker) IsSuccessful() func(err error) bool {
	return cb.isSuccessful
}

func (cb *CircuitBreaker) Counts() Counts {

	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

func (cb *CircuitBreaker) Execute(req func() error, opts ...Option) (err error) {

	var generation uint64
	values := prepareOpts(opts)
	if generation, err = cb.beforeRequest(); err != nil {
		if errors.Is(err, ErrOpenState) && values.fallback != nil {
			if fallBackErr := values.fallback(err); fallBackErr == nil {
				return nil
			}
		}
		return
	}
	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()
	err = req()
	isSuccessful := cb.isSuccessful
	if values.isSuccessful != nil {
		isSuccessful = values.isSuccessful
	}
	successful := isSuccessful(err)
	if !successful && values.fallback != nil {
		err = values.fallback(err)
	}
	cb.afterRequest(generation, successful)
	return
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {

	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	now := time.Now()
	state, generation := cb.currentState(now)
	if state == StateOpen {
		return generation, ErrOpenState
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}
	cb.counts.onRequest()
	return generation, nil
}

func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {

	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}
	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {

	switch state {
	case StateClosed:
		cb.counts.onSuccess()
	case StateHalfOpen:
		cb.counts.onSuccess()
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *CircuitBreaker) onFailure(state State, now time.Time) {

	switch state {
	case StateClosed:
		cb.counts.onFailure()
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {

	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) setState(state State, now time.Time) {

	if cb.state == state {
		return
	}
	prev := cb.state
	cb.state = state
	cb.toNewGeneration(now)
	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {

	cb.generation++
	cb.counts.clear()
	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default:
		cb.expiry = zero
	}
}

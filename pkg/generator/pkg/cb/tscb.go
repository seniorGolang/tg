package cb

type TwoStepCircuitBreaker struct {
	cb *CircuitBreaker
}

func NewTwoStepCircuitBreaker(name string, st Settings) *TwoStepCircuitBreaker {
	return &TwoStepCircuitBreaker{
		cb: NewCircuitBreaker(name, st),
	}
}

func (cbTs *TwoStepCircuitBreaker) Name() string {
	return cbTs.cb.Name()
}

func (cbTs *TwoStepCircuitBreaker) State() State {
	return cbTs.cb.State()
}

func (cbTs *TwoStepCircuitBreaker) Counts() Counts {
	return cbTs.cb.Counts()
}

func (cbTs *TwoStepCircuitBreaker) Allow() (done func(success bool), err error) {

	var generation uint64
	if generation, err = cbTs.cb.beforeRequest(); err != nil {
		return
	}
	done = func(success bool) {
		cbTs.cb.afterRequest(generation, success)
	}
	return
}

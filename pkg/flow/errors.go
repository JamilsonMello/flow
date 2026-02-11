package flow

import (
	"errors"
	"fmt"
)

var (
	ErrFlowNotFound = errors.New("flow: not found")
	ErrFlowSkipped  = errors.New("flow: skipped (production mode)")
	ErrLimitReached = errors.New("flow: execution limit reached")
)

type FlowError struct {
	Op       string
	FlowName string
	Err      error
}

func (e *FlowError) Error() string {
	if e.FlowName != "" {
		return fmt.Sprintf("flow.%s [%s]: %v", e.Op, e.FlowName, e.Err)
	}
	return fmt.Sprintf("flow.%s: %v", e.Op, e.Err)
}

func (e *FlowError) Unwrap() error {
	return e.Err
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrFlowNotFound)
}

func IsSkipped(err error) bool {
	return errors.Is(err, ErrFlowSkipped)
}

func IsLimitReached(err error) bool {
	return errors.Is(err, ErrLimitReached)
}

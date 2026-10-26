package mocks

import mock "github.com/stretchr/testify/mock"

type Orchestrator struct {
	mock.Mock
}

func (_m *Orchestrator) SetDrainRate(_a0 float64) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for SetDrainRate")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(float64) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func NewOrchestrator(t interface {
	mock.TestingT
	Cleanup(func())
}) *Orchestrator {
	mock := &Orchestrator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

package mocks

import (
	entities "github.com/adrianozp/gaardrail/app/entities"
	mock "github.com/stretchr/testify/mock"
)

type ProcessMetrics struct {
	mock.Mock
}

func (_m *ProcessMetrics) Process(m entities.Metrics) error {
	ret := _m.Called(m)

	if len(ret) == 0 {
		panic("no return value specified for Process")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(entities.Metrics) error); ok {
		r0 = rf(m)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func NewProcessMetrics(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProcessMetrics {
	mock := &ProcessMetrics{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

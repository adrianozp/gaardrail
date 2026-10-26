package mocks

import (
	mock "github.com/stretchr/testify/mock"

	time "time"
)

type Controller struct {
	mock.Mock
}

func (_m *Controller) Compute(measured float64, measureTime time.Time) (float64, error) {
	ret := _m.Called(measured, measureTime)

	if len(ret) == 0 {
		panic("no return value specified for Compute")
	}

	var r0 float64
	var r1 error
	if rf, ok := ret.Get(0).(func(float64, time.Time) (float64, error)); ok {
		return rf(measured, measureTime)
	}
	if rf, ok := ret.Get(0).(func(float64, time.Time) float64); ok {
		r0 = rf(measured, measureTime)
	} else {
		r0 = ret.Get(0).(float64)
	}

	if rf, ok := ret.Get(1).(func(float64, time.Time) error); ok {
		r1 = rf(measured, measureTime)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func NewController(t interface {
	mock.TestingT
	Cleanup(func())
}) *Controller {
	mock := &Controller{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

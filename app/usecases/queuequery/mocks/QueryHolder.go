package mocks

import mock "github.com/stretchr/testify/mock"

type QueryHolder struct {
	mock.Mock
}

func (_m *QueryHolder) Query() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Query")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

func (_m *QueryHolder) SetQuery(_a0 string) {
	_m.Called(_a0)
}

func NewQueryHolder(t interface {
	mock.TestingT
	Cleanup(func())
}) *QueryHolder {
	mock := &QueryHolder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

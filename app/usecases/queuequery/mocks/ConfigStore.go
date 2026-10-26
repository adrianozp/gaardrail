package mocks

import mock "github.com/stretchr/testify/mock"

type ConfigStore struct {
	mock.Mock
}

func (_m *ConfigStore) Set(updates map[string]interface{}) error {
	ret := _m.Called(updates)

	if len(ret) == 0 {
		panic("no return value specified for Set")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(map[string]interface{}) error); ok {
		r0 = rf(updates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func NewConfigStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *ConfigStore {
	mock := &ConfigStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

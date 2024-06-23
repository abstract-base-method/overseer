package tests

import "github.com/stretchr/testify/mock"

type mockEventHandler struct {
	mock.Mock
}

func (m *mockEventHandler) Handle(ctx interface{}, event interface{}) (<-chan interface{}, error) {
	ret := m.Called(ctx, event)

	var r0 <-chan interface{}
	if rf, ok := ret.Get(0).(func(interface{}, interface{}) <-chan interface{}); ok {
		r0 = rf(ctx, event)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(interface{}, interface{}) error); ok {
		r1 = rf(ctx, event)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (m *mockEventHandler) Name() string {
	ret := m.Called()
	r0 := ret.String(0)
	return r0
}

func (m *mockEventHandler) Predicate() func(interface{}, interface{}) (bool, error) {
	ret := m.Called()
	var r0 func(interface{}, interface{}) (bool, error)
	if rf, ok := ret.Get(0).(func(interface{}, interface{}) (bool, error)); ok {
		r0 = rf
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(func(interface{}, interface{}) (bool, error))
		}
	}
	return r0
}

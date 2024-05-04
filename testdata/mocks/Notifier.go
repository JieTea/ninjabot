// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocks

import (
	model "github.com/rodrigo-brito/ninjabot/model"
	mock "github.com/stretchr/testify/mock"
)

// Notifier is an autogenerated mock type for the Notifier type
type Notifier struct {
	mock.Mock
}

type Notifier_Expecter struct {
	mock *mock.Mock
}

func (_m *Notifier) EXPECT() *Notifier_Expecter {
	return &Notifier_Expecter{mock: &_m.Mock}
}

// Notify provides a mock function with given fields: _a0
func (_m *Notifier) Notify(_a0 string) {
	_m.Called(_a0)
}

// Notifier_Notify_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Notify'
type Notifier_Notify_Call struct {
	*mock.Call
}

// Notify is a helper method to define mock.On call
//   - _a0 string
func (_e *Notifier_Expecter) Notify(_a0 interface{}) *Notifier_Notify_Call {
	return &Notifier_Notify_Call{Call: _e.mock.On("Notify", _a0)}
}

func (_c *Notifier_Notify_Call) Run(run func(_a0 string)) *Notifier_Notify_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Notifier_Notify_Call) Return() *Notifier_Notify_Call {
	_c.Call.Return()
	return _c
}

func (_c *Notifier_Notify_Call) RunAndReturn(run func(string)) *Notifier_Notify_Call {
	_c.Call.Return(run)
	return _c
}

// OnError provides a mock function with given fields: err
func (_m *Notifier) OnError(err error) {
	_m.Called(err)
}

// Notifier_OnError_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'OnError'
type Notifier_OnError_Call struct {
	*mock.Call
}

// OnError is a helper method to define mock.On call
//   - err error
func (_e *Notifier_Expecter) OnError(err interface{}) *Notifier_OnError_Call {
	return &Notifier_OnError_Call{Call: _e.mock.On("OnError", err)}
}

func (_c *Notifier_OnError_Call) Run(run func(err error)) *Notifier_OnError_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(error))
	})
	return _c
}

func (_c *Notifier_OnError_Call) Return() *Notifier_OnError_Call {
	_c.Call.Return()
	return _c
}

func (_c *Notifier_OnError_Call) RunAndReturn(run func(error)) *Notifier_OnError_Call {
	_c.Call.Return(run)
	return _c
}

// OnOrder provides a mock function with given fields: order
func (_m *Notifier) OnOrder(order model.Order) {
	_m.Called(order)
}

// Notifier_OnOrder_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'OnOrder'
type Notifier_OnOrder_Call struct {
	*mock.Call
}

// OnOrder is a helper method to define mock.On call
//   - order model.Order
func (_e *Notifier_Expecter) OnOrder(order interface{}) *Notifier_OnOrder_Call {
	return &Notifier_OnOrder_Call{Call: _e.mock.On("OnOrder", order)}
}

func (_c *Notifier_OnOrder_Call) Run(run func(order model.Order)) *Notifier_OnOrder_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(model.Order))
	})
	return _c
}

func (_c *Notifier_OnOrder_Call) Return() *Notifier_OnOrder_Call {
	_c.Call.Return()
	return _c
}

func (_c *Notifier_OnOrder_Call) RunAndReturn(run func(model.Order)) *Notifier_OnOrder_Call {
	_c.Call.Return(run)
	return _c
}

// NewNotifier creates a new instance of Notifier. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewNotifier(t interface {
	mock.TestingT
	Cleanup(func())
}) *Notifier {
	mock := &Notifier{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

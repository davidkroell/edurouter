// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/davidkroell/edurouter (interfaces: InternetLayerStrategy)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	edurouter "github.com/davidkroell/edurouter"
	gomock "github.com/golang/mock/gomock"
)

// MockInternetLayerStrategy is a mock of InternetLayerStrategy interface.
type MockInternetLayerStrategy struct {
	ctrl     *gomock.Controller
	recorder *MockInternetLayerStrategyMockRecorder
}

// MockInternetLayerStrategyMockRecorder is the mock recorder for MockInternetLayerStrategy.
type MockInternetLayerStrategyMockRecorder struct {
	mock *MockInternetLayerStrategy
}

// NewMockInternetLayerStrategy creates a new mock instance.
func NewMockInternetLayerStrategy(ctrl *gomock.Controller) *MockInternetLayerStrategy {
	mock := &MockInternetLayerStrategy{ctrl: ctrl}
	mock.recorder = &MockInternetLayerStrategyMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInternetLayerStrategy) EXPECT() *MockInternetLayerStrategyMockRecorder {
	return m.recorder
}

// GetHandler mocks base method.
func (m *MockInternetLayerStrategy) GetHandler(arg0 edurouter.IPProtocol) (edurouter.TransportLayerHandler, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHandler", arg0)
	ret0, _ := ret[0].(edurouter.TransportLayerHandler)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHandler indicates an expected call of GetHandler.
func (mr *MockInternetLayerStrategyMockRecorder) GetHandler(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHandler", reflect.TypeOf((*MockInternetLayerStrategy)(nil).GetHandler), arg0)
}

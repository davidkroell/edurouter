// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/davidkroell/edurouter (interfaces: ARPWriter)

// Package mocks is a generated GoMock package.
package mocks

import (
	net "net"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockARPWriter is a mock of ARPWriter interface.
type MockARPWriter struct {
	ctrl     *gomock.Controller
	recorder *MockARPWriterMockRecorder
}

// MockARPWriterMockRecorder is the mock recorder for MockARPWriter.
type MockARPWriterMockRecorder struct {
	mock *MockARPWriter
}

// NewMockARPWriter creates a new mock instance.
func NewMockARPWriter(ctrl *gomock.Controller) *MockARPWriter {
	mock := &MockARPWriter{ctrl: ctrl}
	mock.recorder = &MockARPWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockARPWriter) EXPECT() *MockARPWriterMockRecorder {
	return m.recorder
}

// SendArpRequest mocks base method.
func (m *MockARPWriter) SendArpRequest(arg0 net.IP) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendArpRequest", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendArpRequest indicates an expected call of SendArpRequest.
func (mr *MockARPWriterMockRecorder) SendArpRequest(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendArpRequest", reflect.TypeOf((*MockARPWriter)(nil).SendArpRequest), arg0)
}

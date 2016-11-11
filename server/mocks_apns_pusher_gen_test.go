// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/smancke/guble/server/apns (interfaces: Pusher)

package server

import (
	gomock "github.com/golang/mock/gomock"
	apns2 "github.com/sideshow/apns2"
)

// Mock of Pusher interface
type MockPusher struct {
	ctrl     *gomock.Controller
	recorder *_MockPusherRecorder
}

// Recorder for MockPusher (not exported)
type _MockPusherRecorder struct {
	mock *MockPusher
}

func NewMockPusher(ctrl *gomock.Controller) *MockPusher {
	mock := &MockPusher{ctrl: ctrl}
	mock.recorder = &_MockPusherRecorder{mock}
	return mock
}

func (_m *MockPusher) EXPECT() *_MockPusherRecorder {
	return _m.recorder
}

func (_m *MockPusher) Push(_param0 *apns2.Notification) (*apns2.Response, error) {
	ret := _m.ctrl.Call(_m, "Push", _param0)
	ret0, _ := ret[0].(*apns2.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockPusherRecorder) Push(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Push", arg0)
}

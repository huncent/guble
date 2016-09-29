package router

import (
	"testing"
	"time"

	"github.com/smancke/guble/protocol"
	"github.com/stretchr/testify/assert"
)

var (
	dummyPath          = protocol.Path("/dummy")
	dummyMessageWithID = &protocol.Message{ID: 1, Path: dummyPath, Body: []byte("dummy body")}
	chanSize           = 10
	queueSize          = 5
)

// Send messages in a zero queued route and expect the route to be closed
// Same test exists for the router
// see router_test.go:TestRoute_IsRemovedIfChannelIsFull
func TestRouteDeliver_sendDirect(t *testing.T) {
	a := assert.New(t)
	r := testRoute()

	for i := 0; i < chanSize; i++ {
		err := r.Deliver(dummyMessageWithID)
		a.NoError(err)
	}

	done := make(chan bool)
	go func() {
		r.Deliver(dummyMessageWithID)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		a.Fail("Message not getting sent!")
	}

	for i := 0; i < chanSize; i++ {
		select {
		case _, open := <-r.MessagesChannel():
			a.True(open)
		case <-time.After(time.Millisecond * 10):
			a.Fail("error not enough messages in channel")
		}
	}

	// and the channel is closed
	select {
	case _, open := <-r.MessagesChannel():
		a.False(open)
	default:
		logger.Debug("len(r.C): %v", len(r.MessagesChannel()))
		a.Fail("channel was not closed")
	}

	a.True(r.invalid)
	a.False(r.consuming)
	a.Equal(0, r.queue.size())
}

func TestRouteDeliver_Invalid(t *testing.T) {
	a := assert.New(t)
	r := testRoute()
	r.invalid = true

	err := r.Deliver(dummyMessageWithID)
	a.Equal(ErrInvalidRoute, err)
}

func TestRouteDeliver_QueueSize(t *testing.T) {
	a := assert.New(t)
	// create a route with a queue size
	r := testRoute()
	r.queueSize = queueSize

	// fill the channel buffer and the queue
	for i := 0; i < chanSize+queueSize; i++ {
		r.Deliver(dummyMessageWithID)
	}

	// and the route should close itself if the queue is overflowed
	done := make(chan bool)
	go func() {
		err := r.Deliver(dummyMessageWithID)
		a.NotNil(err)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(40 * time.Millisecond):
		a.Fail("Message not delivering.")
	}
	time.Sleep(10 * time.Millisecond)
	a.True(r.isInvalid())
	a.False(r.isConsuming())
}

func TestRouteDeliver_WithTimeout(t *testing.T) {
	a := assert.New(t)

	// create a route with timeout and infinite queue size
	r := testRoute()
	r.queueSize = -1 // infinite queue size
	r.timeout = 10 * time.Millisecond

	// fill the channel buffer
	for i := 0; i < chanSize; i++ {
		r.Deliver(dummyMessageWithID)
	}

	// delivering one more message should result in a closed route
	done := make(chan bool)
	go func() {
		err := r.Deliver(dummyMessageWithID)
		a.NoError(err)
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(40 * time.Millisecond):
		a.Fail("Message not delivering.")
	}

	time.Sleep(30 * time.Millisecond)
	err := r.Deliver(dummyMessageWithID)
	a.Equal(ErrInvalidRoute, err)
	a.True(r.invalid)
	a.False(r.consuming)
}

func TestRoute_CloseTwice(t *testing.T) {
	a := assert.New(t)

	r := testRoute()
	err := r.Close()
	a.Equal(ErrInvalidRoute, err)

	err = r.Close()
	a.Equal(ErrInvalidRoute, err)
}

func TestQueue_ShiftEmpty(t *testing.T) {
	q := newQueue(5)
	q.remove()
	assert.Equal(t, 0, q.size())
}

func testRoute() *Route {
	options := RouteConfig{
		RouteParams: RouteParams{
			"application_id": "appID",
			"user_id":        "userID",
		},
		Path:        protocol.Path(dummyPath),
		ChannelSize: chanSize,
	}
	return NewRoute(options)
}

func TestRoute_messageFilter(t *testing.T) {
	a := assert.New(t)

	route := NewRoute(RouteConfig{
		Path:        "/topic",
		ChannelSize: 1,
		RouteParams: RouteParams{
			"field1": "value1",
			"field2": "value2",
		},
	})

	msg := &protocol.Message{
		ID:   1,
		Path: "/topic",
	}
	route.Deliver(msg)

	// test message is received on the channel
	a.True(isMessageReceived(route, msg))

	msg = &protocol.Message{
		ID:   1,
		Path: "/topic",
	}
	msg.SetFilter("field1", "value1")
	route.Deliver(msg)
	a.True(isMessageReceived(route, msg))

	msg = &protocol.Message{
		ID:   1,
		Path: "/topic",
	}
	msg.SetFilter("field1", "value1")
	msg.SetFilter("field2", "value2")
	route.Deliver(msg)
	a.True(isMessageReceived(route, msg))

	msg = &protocol.Message{
		ID:   1,
		Path: "/topic",
	}
	msg.SetFilter("field1", "value1")
	msg.SetFilter("field2", "value2")
	msg.SetFilter("field3", "value3")
	route.Deliver(msg)
	a.False(isMessageReceived(route, msg))

	msg = &protocol.Message{
		ID:   1,
		Path: "/topic",
	}
	msg.SetFilter("field3", "value3")
	route.Deliver(msg)
	a.False(isMessageReceived(route, msg))
}

func isMessageReceived(route *Route, msg *protocol.Message) bool {
	select {
	case m, opened := <-route.MessagesChannel():
		if !opened {
			return false
		}

		return m == msg
	case <-time.After(20 * time.Millisecond):
	}

	return false
}

// Test route fetching mechanism.
// If a route has a fetch request it should return the messages from the store
// and then continue with the messages received from the router.
//
// Based on the fetch request the route may not accept subscription and just close the
// channel when the fetch is done.
func TestRoute_FetchRequest(t *testing.T) {

}

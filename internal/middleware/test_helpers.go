package middleware

import (
	"net/http"

	"connectrpc.com/connect"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
)

type mockRequest struct {
	*connect.Request[v1.ListProjectsRequest]
	procedure string
}

func newMockRequest(procedure string) connect.AnyRequest {
	return &mockRequest{
		Request:   connect.NewRequest(&v1.ListProjectsRequest{}),
		procedure: procedure,
	}
}

func (m *mockRequest) Spec() connect.Spec {
	return connect.Spec{Procedure: m.procedure}
}

type mockStreamingHandlerConn struct {
	procedure string
}

func (m *mockStreamingHandlerConn) Spec() connect.Spec {
	return connect.Spec{Procedure: m.procedure}
}

func (m *mockStreamingHandlerConn) RequestHeader() http.Header {
	return http.Header{}
}

func (m *mockStreamingHandlerConn) Send(interface{}) error {
	return nil
}

func (m *mockStreamingHandlerConn) Receive(interface{}) error {
	return nil
}

func (m *mockStreamingHandlerConn) Close() error {
	return nil
}

func (m *mockStreamingHandlerConn) Peer() connect.Peer {
	return connect.Peer{}
}

func (m *mockStreamingHandlerConn) ResponseHeader() http.Header {
	return http.Header{}
}

func (m *mockStreamingHandlerConn) ResponseTrailer() http.Header {
	return http.Header{}
}

var (
	_ connect.AnyRequest           = (*mockRequest)(nil)
	_ connect.StreamingHandlerConn = (*mockStreamingHandlerConn)(nil)
)

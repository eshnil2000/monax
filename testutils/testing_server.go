package testutils

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// This is a fake server implementation to test Eris command line tools
// hitting proper API endpoints.

// Server holds an httptest.Server handler along with the last recorded
// API endpoint call (path, method, body).
type Server struct {
	mu       *sync.RWMutex
	server   *httptest.Server
	response ServerResponse

	path   string
	method string
	body   string
}

// ServerResponse holds a prerecorded server response for any API call.
// The response can be set with the SetResponse() method.
type ServerResponse struct {
	Code   int
	Body   string
	Header http.Header
}

// ServeHTTP is an http.Handler interface implementation. This handler serves
// all API calls and records the last path, method and request body.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	// Record request.
	s.setMethod(r.Method)
	s.setPath(r.URL.Path)
	s.setBody(string(body))

	// Send out prerecorded response.
	for k, v := range s.response.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(s.response.Code)
	fmt.Fprintf(w, s.response.Body)
}

// NewServer creates a new fake server that serves requests at addr base URL.
// addr parameter is optional; if it is omitted, random port is used to serve
// on localhost. That random port can be retrieved with server.URL() call.
// NewServer panics on error.
//
// Usage:
//
//   server := testutils.NewServer()
//
//   server := testutils.NewServer("localhost:8080")
//   server.SetResponse(testutils.ServerResponse{
//	Code: http.StatusNotFound,
//      Body: "{}",
//      Header: map[string][]string{
//     	   "Content-Type": []string{"application/json"},
//      },
//   })
//
func NewServer(addr ...interface{}) *Server {
	s := &Server{
		mu: &sync.RWMutex{},
		response: ServerResponse{
			Code:   http.StatusOK,
			Header: make(map[string][]string),
		},
	}

	// NewServer().
	if len(addr) == 0 {
		s.server = httptest.NewServer(s)
		return s
	}

	// NewServer(addr).
	s.server = httptest.NewUnstartedServer(s)

	if _, ok := addr[0].(string); !ok {
		panic("can accept only strings as addr")
	}

	listener, err := net.Listen("tcp", addr[0].(string))
	if err != nil {
		// to hedge against the defer statement taking a second
		// to close the connection to the address from any
		// previous tests...
		time.Sleep(3 * time.Second)
		listener, err = net.Listen("tcp", addr[0].(string))
		if err != nil {
			panic(err)
		}
	}
	s.server.Listener = listener
	s.server.Start()

	return s
}

// Method returns the last HTTP method used to call the server.
func (s *Server) Method() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.method
}

// Path returns the last API path used to call the server.
func (s *Server) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.path
}

// Body returns the last request body sent to the server.
func (s *Server) Body() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.body
}

// URL returns the base URL path for the server.
func (s Server) URL() string {
	return s.server.URL
}

// Close stops the server.
func (s *Server) Close() {
	s.server.Close()
}

// SetResponse changes the prerecorded response for the running server.
func (s *Server) SetResponse(response ServerResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.response = response
}

func (s *Server) setMethod(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.method = method
}

func (s *Server) setPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.path = path
}

func (s *Server) setBody(body string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.body = body
}

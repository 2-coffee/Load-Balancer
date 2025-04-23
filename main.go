package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy // reverse proxy allows us to type in domain name instead of ip address
}

// Create a new simple server.
func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	handelErr(err)

	return &simpleServer{ // return a pointer to a new struct
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

// Just initialize a new load balancer.
func newLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

// prints error
func handelErr(err error) {
	if err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func (s *simpleServer) Address() string { return s.addr }

func (s *simpleServer) IsAlive() bool { return true }

func (s *simpleServer) Serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

// Method belonging to Load Balancer type.
func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount]
	for !server.IsAlive() {
		lb.roundRobinCount = (lb.roundRobinCount + 1) % len(lb.servers) // looping through the servers
		server = lb.servers[lb.roundRobinCount]
	}
	lb.roundRobinCount = (lb.roundRobinCount + 1) % len(lb.servers)
	return server
}

func (lb *LoadBalancer) serverProxy(rw http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	log.Printf("forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, r)
}

func main() {
	servers := []Server{ // example servers
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.google.com/search"),
		newSimpleServer("https://www.amazon.com"),
	}
	lb := newLoadBalancer("8080", servers) // pass in our slice of servers
	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serverProxy(rw, r)
	}
	http.HandleFunc("/", handleRedirect)

	log.Printf("serving requests at localhost: %s \n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}

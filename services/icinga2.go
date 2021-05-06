package services

import (
	"crypto/tls"
	"net/http"
)

type Icinga2 interface {
	Node(name string) Icinga2Node
	Cleanup()
}

type Icinga2Node interface {
	Host() string
	Port() string
	Cleanup()
}

type icinga2NodeInfo struct {
	host string
	port string
}

func (r *icinga2NodeInfo) Host() string {
	return r.host
}

func (r *icinga2NodeInfo) Port() string {
	return r.port
}

func Icinga2NodeApiClient(n Icinga2Node) *http.Client {
	return &http.Client{
		Transport: &icinga2NodeApiHttpTransport{
			host:     n.Host() + ":" + n.Port(),
			username: "root", // TODO(jb)
			password: "root", // TODO(jb)
			wrappedTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// TODO(jb): certificate validation
					InsecureSkipVerify: true,
					//ServerName: ...,
					//RootCAs:    ...,
				},
			},
		},
	}
}

type icinga2NodeApiHttpTransport struct {
	host             string
	username         string
	password         string
	wrappedTransport http.RoundTripper
}

func (t *icinga2NodeApiHttpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Host = t.host
	req.URL.Host = t.host
	req.URL.Scheme = "https"
	req.SetBasicAuth(t.username, t.password)
	return t.wrappedTransport.RoundTrip(req)
}

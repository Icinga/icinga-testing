package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

type Icinga2Client struct {
	http.Client
}

func NewIcinga2Client(address string, username string, password string) *Icinga2Client {
	return &Icinga2Client{
		http.Client{
			Transport: &icinga2ClientHttpTransport{
				host:     address,
				username: username,
				password: password,
				wrappedTransport: &http.Transport{
					TLSClientConfig: &tls.Config{
						// TODO(jb): certificate validation
						InsecureSkipVerify: true,
						//ServerName: ...,
						//RootCAs:    ...,
					},
				},
			},
		},
	}
}

func (c *Icinga2Client) addJsonHeaders(req *http.Request) {
	req.Header.Add("Accept", "application/json")
	if req.Body != nil {
		req.Header.Add("Content-Type", "application/json")
	}
}

func (c *Icinga2Client) GetJson(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.addJsonHeaders(req)
	return c.Do(req)
}

func (c *Icinga2Client) PutJson(url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return nil, err
	}
	c.addJsonHeaders(req)
	return c.Do(req)
}

func (c *Icinga2Client) PostJson(url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	c.addJsonHeaders(req)
	return c.Do(req)
}

func (c *Icinga2Client) DeleteJson(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	c.addJsonHeaders(req)
	return c.Do(req)
}

func (c *Icinga2Client) CreateHost(t *testing.T, name string, body interface{}) {
	if body == nil {
		body = map[string]interface{}{
			"attrs": map[string]interface{}{
				"check_command":  "dummy",
			},
		}
	}
	bodyJson, err := json.Marshal(body)
	require.NoError(t, err, "json.Marshal() should succeed")
	res, err := c.PutJson("/v1/objects/hosts/"+name, bytes.NewBuffer(bodyJson))
	require.NoErrorf(t, err, "PUT request for host %s should succeed", name)
	require.Equalf(t, http.StatusOK, res.StatusCode, "PUT for host %s should return OK", name)
}

func (c *Icinga2Client) DeleteHost(t *testing.T, name string, cascade bool) {
	params := ""
	if cascade {
		params = "?cascade=1"
	}
	res, err := c.DeleteJson("/v1/objects/hosts/" + name + params)
	require.NoErrorf(t, err, "DELETE request for host %s should succeed", name)
	require.Equalf(t, http.StatusOK, res.StatusCode, "DELETE for host %s should return OK", name)
}

func (c *Icinga2Client) CreateService(t *testing.T, host string, service string, body interface{}) {
	if body == nil {
		body = map[string]interface{}{
			"attrs": map[string]interface{}{
				"check_command":  "dummy",
			},
		}
	}
	bodyJson, err := json.Marshal(body)
	require.NoError(t, err, "json.Marshal() should succeed")
	res, err := c.PutJson("/v1/objects/services/"+host+"!"+service, bytes.NewBuffer(bodyJson))
	require.NoErrorf(t, err, "PUT request for service %s!%s should succeed", host, service)
	require.Equalf(t, http.StatusOK, res.StatusCode, "PUT for service %s!%s should return OK", host, service)
}

func (c *Icinga2Client) DeleteService(t *testing.T, host string, service string, cascade bool) {
	params := ""
	if cascade {
		params = "?cascade=1"
	}
	res, err := c.DeleteJson("/v1/objects/services/" + host + "!" + service + params)
	require.NoErrorf(t, err, "DELETE for service %s!%s should succeed", host, service)
	require.Equalf(t, http.StatusOK, res.StatusCode, "DELETE for service %s!%s should return OK", host, service)
}

type icinga2ClientHttpTransport struct {
	host             string
	username         string
	password         string
	wrappedTransport http.RoundTripper
}

func (t *icinga2ClientHttpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Host = t.host
	req.URL.Host = t.host
	req.URL.Scheme = "https"
	req.SetBasicAuth(t.username, t.password)
	return t.wrappedTransport.RoundTrip(req)
}

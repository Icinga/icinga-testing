package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"github.com/stretchr/testify/assert"
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

func (c *Icinga2Client) CreateObject(t testing.TB, typ string, name string, body interface{}) {
	bodyJson, err := json.Marshal(body)
	require.NoError(t, err, "json.Marshal() should succeed")
	url := "/v1/objects/" + typ + "/" + name
	res, err := c.PutJson(url, bytes.NewBuffer(bodyJson))
	require.NoErrorf(t, err, "PUT request for %s should succeed", url)
	if !assert.Equalf(t, http.StatusOK, res.StatusCode, "PUT request for %s should return OK", url) {
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err, "reading response for PUT request for %s", url)
		t.Logf("\nAPI response: %s\n\n%s\n\nRequest body:\n\n%s", res.Status, body, bodyJson)
	}
}

func (c *Icinga2Client) UpdateObject(t testing.TB, typ string, name string, body interface{}) {
	bodyJson, err := json.Marshal(body)
	require.NoError(t, err, "json.Marshal() should succeed")
	url := "/v1/objects/" + typ + "/" + name
	res, err := c.PostJson(url, bytes.NewBuffer(bodyJson))
	require.NoErrorf(t, err, "POST request for %s should succeed", url)
	require.Equalf(t, http.StatusOK, res.StatusCode, "POST request for %s should return OK", url)
}

func (c *Icinga2Client) DeleteObject(t testing.TB, typ string, name string, cascade bool) {
	params := ""
	if cascade {
		params = "?cascade=1"
	}
	url := "/v1/objects/" + typ + "/" + name + params
	res, err := c.DeleteJson(url)
	require.NoErrorf(t, err, "DELETE request for %s should succeed", url)
	require.Equalf(t, http.StatusOK, res.StatusCode, "DELETE request for %s should return OK", url)
}

func (c *Icinga2Client) CreateHost(t testing.TB, name string, body interface{}) {
	if body == nil {
		body = map[string]interface{}{
			"attrs": map[string]interface{}{
				"check_command": "dummy",
			},
		}
	}
	c.CreateObject(t, "hosts", name, body)
}

func (c *Icinga2Client) DeleteHost(t testing.TB, name string, cascade bool) {
	c.DeleteObject(t, "hosts", name, cascade)
}

func (c *Icinga2Client) CreateService(t testing.TB, host string, service string, body interface{}) {
	if body == nil {
		body = map[string]interface{}{
			"attrs": map[string]interface{}{
				"check_command": "dummy",
			},
		}
	}
	c.CreateObject(t, "services", host+"!"+service, body)
}

func (c *Icinga2Client) DeleteService(t testing.TB, host string, service string, cascade bool) {
	c.DeleteObject(t, "services", host+"!"+service, cascade)
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

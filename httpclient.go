package rdap

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

//禁用golang http模块的自动跳转301和302
func skipRedirects(req *http.Request, via []*http.Request) error {
	if len(via) > 0 {
		return skipRedirect
	}
	return nil
}

var skipRedirect = errors.New(`stop redirect`)

//client禁用tls证书验证，并自定义超时时间
func NewClient(timeout int) *http.Client {
	var tr http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	var Jar, _ = cookiejar.New(nil)
	client := &http.Client{
		Transport:     tr,
		CheckRedirect: skipRedirects,
		Timeout:       time.Duration(timeout) * time.Second,
		Jar:           Jar,
	}
	return client
}

func SkipRedirect(c *http.Client, r *http.Request) (*http.Response, error) {
	resp, err := c.Do(r)
	if err != nil {
		if ue, ok := err.(*url.Error); ok && ue.Err != skipRedirect {
			return nil, err
		}
	}
	return resp, nil
}

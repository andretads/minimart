package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func New(upstream string, modifyRequest func(*http.Request)) (*httputil.ReverseProxy, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(u)
			pr.SetXForwarded()
			if modifyRequest != nil {
				modifyRequest(pr.Out)
			}
			pr.Out.Host = u.Host
		},
	}, nil
}

package proxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func New(address string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(address)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return nil, errors.New("invalid upstream HTTP service URL")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 60 * time.Second
	transport.IdleConnTimeout = 30 * time.Second
	return &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.SetXForwarded()
		},
		Transport: transport,
		ModifyResponse: func(res *http.Response) error {
			res.Header.Del("Access-Control-Allow-Origin")
			res.Header.Del("Access-Control-Allow-Methods")
			res.Header.Del("Access-Control-Allow-Headers")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "service unavailable", http.StatusBadGateway)
		},
	}, nil
}

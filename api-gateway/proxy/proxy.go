package proxy

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

func NewReverseProxy(targetURL string, stripPrefix string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirectr := proxy.Director

	proxy.Director = func(req *http.Request) {
		originalDirectr(req)

		req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
	}

	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("service unavailable"))
	}

	proxy.ModifyResponse = func(res *http.Response) error {
		res.Header.Del("Access-Control-Allow-Origin")
		res.Header.Del("Access-Control-Allow-Methods")
		res.Header.Del("Access-Control-Allow-Headers")
		return nil
	}

	return func(c *gin.Context) {
		if userID, ok := c.Get("user_id"); ok {
			c.Request.Header.Set("X-User-Id", fmt.Sprint(userID))
		}
		if role, ok := c.Get("role"); ok {
			c.Request.Header.Set("X-User-Role", fmt.Sprint(role))
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

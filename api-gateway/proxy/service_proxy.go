package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func NewServiceProxy(targetURL string, stripPrefix string) gin.HandlerFunc {
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

	return func(c *gin.Context) {
		if userID, ok := c.Get("user_id"); ok {
			c.Request.Header.Set("X-User-Id", fmt.Sprint(userID))
		}
		if assistantID, ok := c.Get("assistant_id"); ok {
			c.Request.Header.Set("X-Assistant-Id", fmt.Sprint(assistantID))
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

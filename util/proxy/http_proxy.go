package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

/**
抽象出代理类
*/

type Proxy interface {
	HttpProxy(w http.ResponseWriter, r *http.Request) error
}

type TrackerProxy struct {
	Schema string
	Addr   string
	Target string
}

func NewTrackerProxy(schema string, addr string) *TrackerProxy {
	return &TrackerProxy{
		Schema: schema,
		Addr:   addr,
		Target: schema + "://" + addr,
	}
}

//HttpProxy tracker请求的反向代理
func (tp *TrackerProxy) HttpProxy(w http.ResponseWriter, r *http.Request) error {
	remote, err := url.Parse(tp.Target)
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	r.URL.Host = remote.Host
	r.URL.Scheme = remote.Scheme
	r.Header.Set("X-Forward-Host", r.Header.Get("Host"))
	r.Host = remote.Host

	proxy.ServeHTTP(w, r)
	return nil
}

package proxy

import "net/http"

type Entity struct {
	Schema string
	Addr   string
	Target string
}

type ErrorHandler func(http.ResponseWriter, *http.Request, error)

//Proxy 反向代理接口
type Proxy interface {
	HttpProxy(w http.ResponseWriter, r *http.Request, handler ErrorHandler) error
}

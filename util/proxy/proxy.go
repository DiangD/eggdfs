package proxy

import "net/http"

//Proxy 反向代理接口
type Proxy interface {
	HttpProxy(w http.ResponseWriter, r *http.Request) error
}

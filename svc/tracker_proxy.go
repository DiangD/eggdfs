package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/svc/proxy"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
)

/**
抽象出代理类
*/

type TrackerProxy struct {
	proxy.Entity
	Group    string
	ClientIP string
	tracker  *Tracker
}

//NewTrackerProxy 构造函数
func NewTrackerProxy(schema, addr, clientIP, group string, tracker *Tracker) *TrackerProxy {
	t := &TrackerProxy{
		Group:    group,
		tracker:  tracker,
		ClientIP: clientIP,
		Entity: proxy.Entity{
			Schema: schema,
			Addr:   addr,
			Target: schema + "://" + addr,
		},
	}
	return t
}

//HttpProxy tracker请求的反向代理 todo
func (tp *TrackerProxy) HttpProxy(w http.ResponseWriter, r *http.Request, handler proxy.ErrorHandler) error {
	remote, err := url.Parse(tp.Target)
	if err != nil {
		return err
	}
	rp := httputil.NewSingleHostReverseProxy(remote)
	r.URL.Host = remote.Host
	r.URL.Scheme = remote.Scheme
	r.Header.Set("X-Forward-Host", r.Header.Get("Host"))
	r.Host = remote.Host

	//502回调
	//handler could be nil
	if handler != nil {
		rp.ErrorHandler = handler
	}
	rp.ServeHTTP(w, r)
	return nil
}

func (tp *TrackerProxy) AbortErrorHandler(w http.ResponseWriter, req *http.Request, err error) {
	s := tp.tracker.groups[tp.Group].GetStorage(req.Host)
	s.Status = common.StorageOffline

	res := model.RespResult{
		Status:  common.Fail,
		Message: err.Error(),
	}
	bytes, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_, _ = w.Write(bytes)
}

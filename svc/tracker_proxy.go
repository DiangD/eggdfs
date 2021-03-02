package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/svc/proxy"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
	"net/url"
)

/**
//todo 需要重构整个代理实现
抽象出代理类
*/

type TrackerProxy struct {
	proxy.Entity
	Group   string
	tracker *Tracker
	c       *gin.Context
}

//NewTrackerProxy 构造函数
func NewTrackerProxy(schema, addr, group string, tracker *Tracker, c *gin.Context) *TrackerProxy {
	t := &TrackerProxy{
		Group:   group,
		tracker: tracker,
		Entity: proxy.Entity{
			Schema: schema,
			Addr:   addr,
			Target: schema + "://" + addr,
		},
		c: c,
	}
	return t
}

//HttpProxy tracker请求的反向代理
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

//AbortErrorHandler 错误处理机制，直接返回
func (tp *TrackerProxy) AbortErrorHandler(w http.ResponseWriter, req *http.Request, err error) {
	group := tp.tracker.GetGroup(tp.Group)
	group.GetStorage(tp.Addr).Status = common.StorageOffline

	res := model.RespResult{
		Status:  common.ProxyBadGateWay,
		Message: err.Error(),
	}
	bytes, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_, _ = w.Write(bytes)
}

package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/logger"
	"eggdfs/util"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"hash/crc32"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Hash func([]byte) uint32

type Tracker struct {
	groups map[string]*Group
	db     *model.EggDB
	hash   Hash
	mu     sync.RWMutex //考虑线程安全
}

//NewTracker 构造函数可使用自定义的hash
func NewTracker(fn Hash) *Tracker {
	t := &Tracker{
		groups: make(map[string]*Group),
		db:     model.NewEggDB("tracker"),
		hash:   fn,
	}
	if t.hash == nil {
		t.hash = crc32.ChecksumIEEE
	}
	return t
}

//Start 服务启动入口
func (t *Tracker) Start() {
	r := gin.Default()

	r.POST("/status", t.StorageStatusReport)
	r.Group("/v1")
	{
		r.POST("/upload", t.QuickUpload)
	}

	err := r.Run(":" + config().Port)
	if err != nil {
		addr := net.JoinHostPort(config().Host, config().Port)
		logger.Panic("Tracker服务启动失败", zap.String("addr", addr))
	}
}

//startTrackerTimerTask tracker定时任务
func (t *Tracker) startTrackerTimerTask() error {
	cr := cron.New(cron.WithSeconds())

	cr.Start()
	return nil
}

//StorageStatusReport 处理storage回报的信息
func (t *Tracker) StorageStatusReport(c *gin.Context) {
	var params struct {
		Group string `json:"group" binding:"required"`
		Host  string `json:"host" binding:"required"`
		Port  string `json:"port" binding:"required"`
		Free  uint64 `json:"free" binding:"required"`
	}
	if err := c.ShouldBindJSON(&params); err != nil {
		logger.Error("param binding fail", zap.String("url", "/status"))
	}

	if params.Port == "" || params.Host == "" {
		logger.Error("nil addr", zap.String("url", "/status"))
		return
	}

	sm := &StorageServer{
		Group:      params.Group,
		Addr:       net.JoinHostPort(params.Host, params.Port),
		Free:       params.Free,
		Status:     common.StorageActive,
		UpdateTime: time.Now().Unix(),
	}

	//group未注册
	if !t.IsExistGroup(sm.Group) {
		group := &Group{
			Name:     sm.Group,
			Status:   common.GroupActive,
			Cap:      sm.Free,
			Storages: make(map[string]*StorageServer),
		}
		_ = group.SaveOrUpdateStorage(sm)
		_ = t.RegisterGroup(group)
	}

	//group已注册
	group := t.groups[sm.Group]
	if err := group.SaveOrUpdateStorage(sm); err != nil {
		group.Cap = group.GetGroupCap()
	}
	logger.Info("tracker group details", zap.Any("groups", t.groups))
	//todo 思路：应该先添加storage还是先计算cap... 逻辑：是否及时计算cap，以及group状态
}

//SelectStorageIPHash ip hash选择storage
func (t *Tracker) SelectStorageIPHash(ip string, g *Group) (s *StorageServer, err error) {
	as := make([]*StorageServer, 0)
	for _, g := range g.GetStorages() {
		if g.Status == common.StorageActive {
			as = append(as, g)
		}
	}
	if len(as) == 0 {
		return nil, errors.New("no available storage for group")
	}
	//ip hash
	hashcode := int(t.hash([]byte(ip)))
	return as[hashcode%len(as)], err
}

//IsExistGroup group是否注册
func (t *Tracker) IsExistGroup(groupName string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.groups[groupName]
	return ok
}

//RegisterGroup 注册group
func (t *Tracker) RegisterGroup(g *Group) error {
	if g.Name == "" {
		return errors.New("group name can not be nil")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.groups[g.Name]; !ok {
		t.groups[g.Name] = g
	}
	return nil
}

//SelectGroupForUpload 选择上传的group
func (t *Tracker) SelectGroupForUpload() (*Group, error) {
	gs := make([]*Group, 0)
	t.mu.RLock()
	for _, g := range t.groups {
		if g.Status == common.GroupActive {
			gs = append(gs, g)
		}
	}
	t.mu.RUnlock()
	if len(gs) == 0 {
		return nil, errors.New("no available group")
	}

	//按容量降序
	sort.Slice(gs, func(i, j int) bool {
		return gs[i].Cap > gs[i].Cap
	})
	//返回容量最大的group
	return gs[0], nil
}

func (t *Tracker) QuickUpload(c *gin.Context) {
	//获取group
	group, err := t.SelectGroupForUpload()
	if err != nil {
		logger.Error(err.Error())
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.Fail,
			Message: err.Error(),
		})
		return
	}
	//获取storage
	s, err := t.SelectStorageIPHash(c.ClientIP(), group)
	if err != nil {
		logger.Error(err.Error())
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.Fail,
			Message: err.Error(),
		})
		return
	}

	//set header
	c.Request.Header.Set(common.HeaderFileUUID, util.GenFileUUID())

	//反向代理
	p := NewTrackerProxy(config().HttpSchema, s.Addr, c.ClientIP(), group.Name, t)
	if err = t.httpProxy(p, c); err != nil {
		logger.Error(err.Error())
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.Fail,
			Message: err.Error(),
		})
		return
	}

}

//httpProxy tracker 反向代理 //todo 处理502请求2 redo proxy
func (t *Tracker) httpProxy(tp *TrackerProxy, c *gin.Context) error {
	return tp.HttpProxy(c.Writer, c.Request, tp.AbortErrorHandler)
}

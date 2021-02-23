package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/logger"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"hash/crc32"
	"sort"
	"time"
)

type Hash func([]byte) uint32

type Tracker struct {
	groups map[string]Group
	db     *model.EggDB
	hash   Hash
}

func NewTracker(fn Hash) *Tracker {
	t := &Tracker{
		groups: make(map[string]Group),
		db:     model.NewEggDB("tracker"),
		hash:   fn,
	}
	if fn == nil {
		t.hash = crc32.ChecksumIEEE
	}
	return t
}

func (t *Tracker) Start() {
	r := gin.Default()

	err := r.Run(config().Port)
	if err != nil {
		logger.Panic("Tracker服务启动失败", zap.String("addr", config().Host))
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
		Group  string `json:"group"`
		Host   string `json:"host"`
		Port   string `json:"port"`
		Free   uint64 `json:"free"`
		Active bool   `json:"active"`
	}
	if err := c.ShouldBind(&params); err != nil {
		logger.Error("param binding fail", zap.String("url", "/status"))
	}

	if params.Port == "" || params.Host == "" {
		logger.Error("nil addr", zap.String("url", "/status"))
		return
	}

	sm := StorageServer{
		Group:      params.Group,
		Addr:       params.Host + params.Port,
		Free:       params.Free,
		UpdateTime: time.Now().Unix(),
	}

	//group未注册
	if !t.IsExistGroup(sm.Group) {
		group := Group{
			Name:     sm.Group,
			Status:   common.GroupActive,
			Cap:      sm.Free,
			Storages: make(map[string]StorageServer),
		}
		_ = group.SaveOrUpdateStorage(sm)
		_ = t.RegisterGroup(group)
	}

	//group已注册
	group := t.groups[sm.Group]
	if err := group.SaveOrUpdateStorage(sm); err != nil {
		group.Cap = group.GetGroupCap()
	}
	//todo 思路：应该先添加storage还是先计算cap...
}

//SelectStorageIPHash ip hash选择storage
func (t *Tracker) SelectStorageIPHash(ip string, g *Group) (s StorageServer, err error) {
	as := make([]StorageServer, 0)
	for _, g := range g.GetStorages() {
		if g.Status == common.StorageActive {
			as = append(as, g)
		}
	}
	if len(as) == 0 {
		return StorageServer{}, errors.New("no available storage for group")
	}
	//ip hash
	hashcode := int(t.hash([]byte(ip)))
	return as[hashcode%len(as)], err
}

//IsExistGroup group是否注册
func (t *Tracker) IsExistGroup(groupName string) bool {
	_, ok := t.groups[groupName]
	return ok
}

//RegisterGroup 注册group
func (t *Tracker) RegisterGroup(g Group) error {
	if g.Name == "" {
		return errors.New("group name can not be nil")
	}
	if _, ok := t.groups[g.Name]; !ok {
		t.groups[g.Name] = g
	}
	return nil
}

//SelectGroupForUpload 选择上传的group
func (t *Tracker) SelectGroupForUpload() (Group, error) {
	gs := make([]Group, 0)
	for _, g := range t.groups {
		if g.Status == common.GroupActive {
			gs = append(gs, g)
		}
	}
	if len(gs) == 0 {
		return Group{}, errors.New("no available group")
	}

	//按容量降序
	sort.Slice(gs, func(i, j int) bool {
		return gs[i].Cap > gs[i].Cap
	})
	//返回容量最大的group
	return gs[0], nil
}

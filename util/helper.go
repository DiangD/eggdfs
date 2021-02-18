package util

import (
	"crypto/md5"
	"eggdfs/logger"
	"eggdfs/svc/conf"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"io"
	"math/rand"
	"path"
	"strconv"
	"strings"
	"time"
)

//DNode 全局雪花算法Node
var DNode *snowflake.Node

func init() {
	GenTrackerNodeId()
}

//GenTrackerNodeId 生成唯一Tracker id
func GenTrackerNodeId() {
	var err error
	if conf.Config().Tracker.NodeId > 0 {
		DNode, err = snowflake.NewNode(conf.Config().Tracker.NodeId)
	}
	rand.Seed(time.Now().UnixNano())
	DNode, err = snowflake.NewNode(int64(rand.Intn(1024)))
	if err != nil {
		logger.Panic("Error Tracker NodeId")
	}
}

//GenFilePath 生成文件路径
func GenFilePath(paths ...string) (root string) {
	var path string
	if len(paths) > 0 {
		path = paths[0]
	}
	cur := time.Now()
	year, month, day := cur.Year(), cur.Month(), cur.Day()
	root = strings.Join([]string{strconv.Itoa(year),
		strconv.Itoa(int(month)),
		strconv.Itoa(day),
	}, "/")
	if path != "" {
		root = root + "/" + path
	}
	return root
}

//GenUUIDFileName 生成文件名
func GenFileUUID() string {
	return DNode.Generate().String()
}

func GenFileName(id, fn string) (fileName string) {
	return id + "." + path.Ext(fn)
}

//GenFileMD5 生成文件md5 不适合大文件
func GenFileMD5(file io.Reader) (string, error) {
	md5h := md5.New()
	if _, err := io.Copy(md5h, file); err != nil {
		return "", err
	}
	md := md5h.Sum(nil)
	return fmt.Sprintf("%x", md), nil
}

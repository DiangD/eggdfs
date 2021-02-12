package util

import (
	"crypto/md5"
	"eggdfs/logger"
	"eggdfs/svc/conf"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var DNode *snowflake.Node

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

func GenUUIDFileName(extension string) (fileName string) {
	fileName = DNode.Generate().String() + "." + extension
	return
}

func GenFileMD5(file io.Reader) (string, error) {
	md5h := md5.New()
	if _, err := io.Copy(md5h, file); err != nil {
		return "", err
	}
	md := md5h.Sum(nil)
	return fmt.Sprintf("%x", md), nil
}

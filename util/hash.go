package util

import (
	"crypto/md5"
)

//自定义hash函数
//MD5Hash 简单利用md5算法的hashcode生成函数
func MD5Hash(src []byte) uint32 {
	hash := md5.New()
	hash.Write(src)
	sum := hash.Sum(nil)
	return uint32(sum[len(sum)-1])
}

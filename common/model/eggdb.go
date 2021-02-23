package model

import (
	"eggdfs/logger"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"sync"
)

const defaultDBDir = "./data"

type EggDB struct {
	Name string
	Ldb  *leveldb.DB
	rm   sync.RWMutex //todo 可能需要读写锁
}

func NewEggDB(name string) *EggDB {
	var err error
	db := &EggDB{
		Name: name,
	}
	path := fmt.Sprintf("%s/%s", defaultDBDir, name)
	db.Ldb, err = leveldb.OpenFile(path, &opt.Options{
		CompactionTableSize: 1024 * 1024 * 20,
		WriteBuffer:         1024 * 1024 * 20,
	})
	if err != nil {
		logger.Panic(fmt.Sprintf("open db file %s fail", path))
	}
	return db
}

func (db *EggDB) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, errors.New("key can not be empty")
	}
	return db.Ldb.Get([]byte(key), nil)
}

func (db *EggDB) Delete(keys ...string) error {
	if len(keys) > 0 {
		for _, key := range keys {
			err := db.Ldb.Delete([]byte(key), nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (db *EggDB) Put(key string, value []byte) error {
	if key == "" {
		return errors.New("key can not be empty")
	}
	err := db.Ldb.Put([]byte(key), value, nil)
	return err
}

func (db *EggDB) IsExistKey(key string) (bool, error) {
	return db.Ldb.Has([]byte(key), nil)
}

package data

import (
	"fmt"
	"github.com/daicheng123/kubejump/config"
	"github.com/patrickmn/go-cache"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"k8s.io/klog/v2"
	"log"
	"sync"
	"time"
)

var (
	DefaultData *Data
	dataOnce    sync.Once
)

func init() {
	InitData()
}
func InitData() *Data {
	if DefaultData == nil {
		dataOnce.Do(func() {
			memCache, _ := newLocalCache()

			db, err := newDB()
			if err != nil {
				klog.Fatal(err.Error())
			}

			DefaultData = newData(memCache, db)
		})
	}
	return DefaultData
}

type Data struct {
	DB         *gorm.DB
	LocalCache *cache.Cache
}

func newData(memCache *cache.Cache, db *gorm.DB) *Data {
	data := new(Data)

	data.LocalCache = memCache

	data.DB = db

	return data
}

func (data *Data) Clean() {
	var (
		localCacheFile = config.GetConf().LocalCachePath
	)

	klog.Info("now, try to save cache file to %s", localCacheFile)
	if err := data.LocalCache.SaveFile(localCacheFile); err != nil {
		log.Println(err)
	}

	klog.Info("now, try to close db connection")
	sqlDB, _ := data.DB.DB()
	sqlDB.Close()
}

func newDB() (db *gorm.DB, err error) {
	conf := config.GetConf()
	dsn := fmt.Sprintf(
		`%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local`,
		conf.DatabaseUser,
		conf.DatabasePassword,
		conf.DatabaseAddress,
		conf.DatabasePort,
		conf.DatabaseName)

	logLevel := gormLogger.Warn

	if config.GetConf().ServerDebug {
		logLevel = gormLogger.Info
	}

	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(logLevel),
	})

	if err != nil {
		klog.Errorf("open database connection failed, err:[%s]", err.Error())
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		klog.Errorf("failed to get db instance failed, err:[%s]", err.Error())
		return
	}
	sqlDB.SetConnMaxIdleTime(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	return
}

func newLocalCache() (*cache.Cache, error) {
	memCache := cache.New(time.Second*36000, time.Second*500)
	localCacheFile := config.GetConf().LocalCachePath
	if len(config.GetConf().LocalCachePath) > 0 {
		//cacheDirectory := filepath.Dir(localCacheFile)
		//
		//err := os.Mkdir(cacheDirectory, os.ModePerm)
		//if err != nil {
		//	klog.Errorf("create cache dir failed: %s", err.Error())
		//}

		if err := memCache.LoadFile(localCacheFile); err != nil {
			klog.Errorf("load cache from dir failed: %s", err.Error())
		}

		go func() {
			ticker := time.Tick(time.Minute)
			for range ticker {
				if err := memCache.SaveFile(config.GetConf().LocalCachePath); err != nil {
					klog.Errorf("load cache from dir failed: %s", err.Error())
				}
			}
		}()
	}
	return memCache, nil
}

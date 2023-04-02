package kubernetes

import (
	"github.com/daicheng123/kubejump/internal/entity"
	lru "github.com/hashicorp/golang-lru"
	workerpool "github.com/panjf2000/ants/v2"
	"k8s.io/klog/v2"
	"os"
	"sync"
)

var (
	DefaultClientFactory *ClientFactory
	factory              *ClientFactory
	mutex                sync.Mutex
)

func init() {
	var err error
	DefaultClientFactory, err = GetClientFactory()
	if err != nil || DefaultClientFactory == nil {
		klog.Errorf("获取K8S客户端工厂失败: %s", err.Error())
		os.Exit(2)
	}
}

const (
	defaultClientSize    = 50
	defaultEvictPoolSize = 50
)

func GetClientFactory() (*ClientFactory, error) {
	if factory != nil {
		return factory, nil
	}
	mutex.Lock()
	defer mutex.Unlock()
	if factory != nil {
		return factory, nil
	}
	var err error
	if err = InitClientFactory(defaultClientSize, defaultEvictPoolSize); err != nil {
		factory = nil
		return nil, err
	}
	return factory, nil
}

type ClientFactory struct {
	lock        sync.Mutex
	pool        *workerpool.Pool
	clientCache *lru.Cache
}

func (cf *ClientFactory) GetOrCreateClient(config *entity.ClusterConfig) (*ClientSet, error) {
	cf.lock.Lock()
	defer cf.lock.Unlock()

	uniqeKey := config.ClientUniqKey()
	client, ok := cf.clientCache.Get(uniqeKey)
	if ok {
		return client.(*ClientSet), nil
	}

	newClient, err := CreateClientSet(config.MasterUrl, config.CaData, config.BearerToken)
	if err != nil {
		return nil, err
	}
	cf.clientCache.Add(uniqeKey, newClient)

	err = cf.pool.Submit(func() {
		cf.freeClientSet(config)
	})

	if err != nil {
		klog.Warning("清除旧版本Client任务提交失败, 等待LRU自动驱逐", err)
	}
	return newClient, nil
}

func InitClientFactory(clientSize int, evictPoolSize int) error {
	cache, err := lru.NewWithEvict(clientSize, func(key interface{}, value interface{}) {
		klog.Infof("集群信息%s已过期", key)
	})

	if err != nil {
		return err
	}

	pool, err := workerpool.NewPool(evictPoolSize)

	if err != nil {
		return err
	}

	factory = &ClientFactory{
		clientCache: cache,
		pool:        pool,
	}
	return nil
}

func CreateClientSet(masterUrl, caData, bearerToken string) (*ClientSet, error) {
	clientSet := &ClientSet{}
	if err := clientSet.initClientSet(masterUrl, caData, bearerToken); err != nil {
		return nil, err
	}
	return clientSet, nil
}

func (cf *ClientFactory) Close() {
	cf.lock.Lock()
	defer cf.lock.Unlock()
	cf.clientCache.Purge()
}

func (cf *ClientFactory) freeClientSet(config *entity.ClusterConfig) {
	oldClientKey := config.OldClientUniqKey()
	if len(oldClientKey) == 0 {
		return
	}
	cf.clientCache.Remove(oldClientKey)
}

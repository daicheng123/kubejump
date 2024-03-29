package kubernetes

import (
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

const (
	POD_INFORMER_NAME       = "pods"
	NAMESPACE_INFORMER_NAME = "namespaces"
)

var (
	informerFactory *InformerFactory
	factoryOnce     sync.Once
)

func GetInformerFactory(cliFactory *ClientFactory) *InformerFactory {
	if informerFactory == nil {
		factoryOnce.Do(func() {
			informerFactory = newInformerFactory(cliFactory)
		})
	}
	return informerFactory
}

func newInformerFactory(cliFactory *ClientFactory) *InformerFactory {

	factory := &InformerFactory{
		cliFactory:  cliFactory,
		lock:        sync.RWMutex{},
		informerMap: make(map[string]InformerInterface, 0),
	}

	return factory
}

type InformerFactory struct {
	cliFactory           *ClientFactory
	lock                 sync.RWMutex
	informerMap          map[string]InformerInterface
	sharedInformerFactor informers.SharedInformerFactory
	//err                  error
}

func (f *InformerFactory) AddInformer(informerKind string, handler cache.ResourceEventHandler, kconfig *entity.ClusterConfig) (*InformerFactory, error) {
	var (
		cli *ClientSet
		err error
	)

	cli, err = f.cliFactory.GetOrCreateClient(kconfig)
	if err != nil {
		return f, err
	}
	uniqueKey := utils.StringBuild(kconfig.ClusterName + "_" + informerKind)

	f.lock.RLock()
	defer f.lock.RUnlock()

	if _, ok := f.informerMap[uniqueKey]; !ok {
		informer := NewInformer(informerKind, handler, cli)
		if informer != nil {
			klog.Infof("informer %s prepare to start", uniqueKey)
			go informer.start()
			f.informerMap[informerKind] = informer
		}
	}
	return f, err
}

func (f *InformerFactory) DelInformer(informerKind string, kconfig *entity.ClusterConfig) *InformerFactory {
	f.lock.RLock()
	defer f.lock.RUnlock()
	uniqueKey := utils.StringBuild(kconfig.ClusterName + "_" + informerKind)

	if informer, ok := f.informerMap[uniqueKey]; ok {
		klog.Infof("informer %s prepare to stop", uniqueKey)
		informer.close()
	}
	delete(f.informerMap, uniqueKey)
	return f
}

func (f *InformerFactory) Close() {
	f.lock.RLock()
	defer f.lock.RUnlock()

	for _, informer := range f.informerMap {
		informer.close()
	}
}

func NewInformer(informerKind string, handler cache.ResourceEventHandler, client *ClientSet) InformerInterface {
	switch informerKind {
	case POD_INFORMER_NAME:
		return newPodInformer(handler, client)
	case NAMESPACE_INFORMER_NAME:
		return newNamespaceInformer(handler, client)
	default:
		return nil
	}
}

type InformerInterface interface {
	start()
	close()
}

type CommonInformer struct {
	InformerInterface
	handler      cache.ResourceEventHandler
	client       *ClientSet
	informer     cache.Controller
	informerChan chan struct{}
}

func (pi *CommonInformer) start() {
	pi.InformerInterface.start()
}

func (pi *CommonInformer) close() {
	close(pi.informerChan)
}

type PodInformer struct {
	*CommonInformer
}

func newPodInformer(handler cache.ResourceEventHandler, cli *ClientSet) *PodInformer {
	var podInformer = new(PodInformer)

	podInformer.CommonInformer = &CommonInformer{
		InformerInterface: podInformer,
		handler:           handler,
		client:            cli,
		informerChan:      make(chan struct{}),
	}

	return podInformer
}

func (pi *PodInformer) start() {
	listWatcher := cache.NewListWatchFromClient(
		pi.client.K8sClientSet.CoreV1().RESTClient(), POD_INFORMER_NAME, metav1.NamespaceAll, fields.Everything())

	_, pi.informer = cache.NewInformer(listWatcher, &corev1.Pod{}, time.Minute*5, pi.handler)

	pi.informer.Run(pi.informerChan)
}

type NamespaceInformer struct {
	*CommonInformer
}

func newNamespaceInformer(handler cache.ResourceEventHandler, cli *ClientSet) *NamespaceInformer {
	var namespaceInformer = new(NamespaceInformer)

	namespaceInformer.CommonInformer = &CommonInformer{
		InformerInterface: namespaceInformer,
		handler:           handler,
		client:            cli,
		informerChan:      make(chan struct{}),
	}

	return namespaceInformer
}

func (pi *NamespaceInformer) start() {
	listWatcher := cache.NewListWatchFromClient(
		pi.client.K8sClientSet.CoreV1().RESTClient(), NAMESPACE_INFORMER_NAME, metav1.NamespaceAll, fields.Everything())

	_, pi.informer = cache.NewInformer(listWatcher, &corev1.Namespace{}, time.Minute*5, pi.handler)

	pi.informer.Run(pi.informerChan)
}

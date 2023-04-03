package service

import (
	"context"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/kubernetes"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
)

type KubernetesService struct {
	*kubeHandlerServices
	clientFactory   *kubernetes.ClientFactory
	informerFactory *kubernetes.InformerFactory
}

func NewKubernetesService() (*KubernetesService, error) {
	clientFactory, err := kubernetes.GetClientFactory()
	informerFactory := kubernetes.GetInformerFactory(clientFactory)
	if err != nil {
		return nil, err
	}

	handlerServices := &kubeHandlerServices{
		handlers:     sync.Map{},
		nsEventChan:  make(chan *nsEvent, 0),
		podEventChan: make(chan *podEvent, 0),
	}

	go func() {
		handlerServices.handlerLoop()
	}()

	return &KubernetesService{clientFactory: clientFactory, informerFactory: informerFactory,
		kubeHandlerServices: handlerServices,
	}, err
}

func (ks *KubernetesService) AddSyncResourceToStore(informerKind string, kconfig *entity.ClusterConfig) (err error) {
	handler := ks.kubeHandlerServices.newHandler(informerKind, kconfig.ClientUniqKey())
	ks.informerFactory, err = ks.informerFactory.AddInformer(informerKind, handler, kconfig)
	return err
}

func (ks *KubernetesService) DelSyncResourceToStore(informerKind string, kconfig *entity.ClusterConfig) {
	ks.informerFactory = ks.informerFactory.DelInformer(informerKind, kconfig)
	ks.kubeHandlerServices.delHandler(informerKind, kconfig.ClientUniqKey())
}

func (ks *KubernetesService) ReloadSyncResourceToStore(informerKind string, kconfig *entity.ClusterConfig) error {
	ks.DelSyncResourceToStore(informerKind, kconfig)
	return ks.AddSyncResourceToStore(informerKind, kconfig)
}

func (ks *KubernetesService) ListNamespaces(ctx context.Context, kconfig *entity.ClusterConfig) (*v1.NamespaceList, error) {
	cli, err := ks.clientFactory.GetOrCreateClient(kconfig)
	if err != nil {
		return nil, err
	}
	return cli.K8sClientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
}

func (ks *KubernetesService) ListPodsByNamespace(ctx context.Context, ns string, kconfig *entity.ClusterConfig) (*v1.PodList, error) {
	cli, err := ks.clientFactory.GetOrCreateClient(kconfig)
	if err != nil {
		return nil, err
	}
	return cli.K8sClientSet.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
}

package service

import (
	"context"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/kubernetes"
	"github.com/toolkits/pkg/container/list"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
)

type KubernetesService struct {
	*kubeHandlerServices
	clientFactory   *kubernetes.ClientFactory
	informerFactory *kubernetes.InformerFactory
}

func NewKubernetesService(podRepo entity.PodRepo, nsRepo entity.NamespaceRepo) (*KubernetesService, error) {
	clientFactory, err := kubernetes.GetClientFactory()
	informerFactory := kubernetes.GetInformerFactory(clientFactory)
	if err != nil {
		return nil, err
	}

	handlerServices := &kubeHandlerServices{
		podRepo:          podRepo,
		eventConcurrency: 10,
		nsRepo:           nsRepo,
		handlers:         sync.Map{},
		eventQueue:       list.NewSafeListLimited(2000),
	}

	go func() {
		handlerServices.LoopHandler(context.Background())
	}()

	return &KubernetesService{clientFactory: clientFactory, informerFactory: informerFactory,
		kubeHandlerServices: handlerServices,
	}, err
}

func (ks *KubernetesService) AddSyncResourceToStore(informerKind string, kconfig *entity.ClusterConfig) (err error) {
	handler := ks.kubeHandlerServices.newHandler(informerKind, kconfig.UniqKey)
	ks.informerFactory, err = ks.informerFactory.AddInformer(informerKind, handler, kconfig)
	return err
}

func (ks *KubernetesService) DelSyncResourceToStore(informerKind string, kconfig *entity.ClusterConfig) {
	ks.informerFactory = ks.informerFactory.DelInformer(informerKind, kconfig)
	ks.kubeHandlerServices.delHandler(informerKind, kconfig.UniqKey)
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

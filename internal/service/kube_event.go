package service

import (
	"context"
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/kubernetes/pods"
	"github.com/daicheng123/kubejump/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

const (
	EVENT_TYPE_ADD    = "ADD"
	EVENT_TYPE_UPDATE = "UPDATE"
	EVENT_TYPE_DELETE = "DELETE"
)

//kubeHandlerServices  handler kubernetes event
type kubeHandlerServices struct {
	handlers     sync.Map
	podRepo      entity.PodRepo
	nsRepo       entity.NamespaceRepo
	podEventChan chan *podEvent
	nsEventChan  chan *nsEvent
}

func (srv *kubeHandlerServices) newHandler(kind string, clusterID uint) *kubeHandler {
	handler := &kubeHandler{
		clusterID: clusterID,
		//clusterUniqKey: uk,
		resourceKind: kind,
		podsChan:     srv.podEventChan,
		nsChan:       srv.nsEventChan,
	}
	srv.handlers.Store(srv.buildKey(handler.resourceKind, handler.clusterID), handler)
	return handler
}

func (srv *kubeHandlerServices) buildKey(kind string, clusterID uint) string {
	return fmt.Sprintf("%s_%d", kind, clusterID)
}

func (srv *kubeHandlerServices) loadHandler(kind string, clusterID uint) *kubeHandler {
	var handler *kubeHandler
	srv.handlers.Range(func(key, value any) bool {
		if key == srv.buildKey(kind, clusterID) {
			handler = value.(*kubeHandler)
			return true
		}
		return false
	})
	return handler
}

func (srv *kubeHandlerServices) delHandler(kind string, clusterID uint) {
	srv.handlers.Delete(srv.buildKey(kind, clusterID))
}

func (srv *kubeHandlerServices) applyPodResource(ctx context.Context, event *podEvent) error {
	return srv.podRepo.CreateOrUpdatePod(ctx, event.Pod)
}

func (srv *kubeHandlerServices) deletePodResource(ctx context.Context, event *podEvent) error {
	return srv.podRepo.DeletePodByNameAndNamespace(ctx, event.PodName, event.Namespace, event.ClusterRef)
}

func (srv *kubeHandlerServices) applyNsResource(ctx context.Context, event *nsEvent) error {
	return srv.nsRepo.CreateOrUpdateNS(ctx, event.Namespace)
}

func (srv *kubeHandlerServices) deleteNsResource(ctx context.Context, event *nsEvent) error {
	return srv.nsRepo.DeleteNSByName(ctx, event.NamespaceName, event.ClusterUniqKey)
}

func (srv *kubeHandlerServices) handlerLoop(ctx context.Context) {
	for {
		select {
		case pe := <-srv.podEventChan:
			if pe.eventType == EVENT_TYPE_ADD || pe.eventType == EVENT_TYPE_UPDATE {

				go utils.RunSafe(func() error {
					klog.Infof("add or update pod %s to storage", pe.PodName)
					return srv.applyPodResource(ctx, pe)
				}, "sync pod add or update event failed.")
			} else if pe.eventType == EVENT_TYPE_DELETE {

				go utils.RunSafe(func() error {
					klog.Infof("delete pod %s to storage", pe.PodName)
					return srv.deletePodResource(ctx, pe)
				}, "sync pod delete event failed.")
			}
		case ne := <-srv.nsEventChan:
			if ne.eventType == EVENT_TYPE_ADD || ne.eventType == EVENT_TYPE_UPDATE {

				go utils.RunSafe(func() error {
					klog.Infof("add or update ns %s to storage", ne.NamespaceName)
					return srv.applyNsResource(ctx, ne)
				}, "sync namespace add or update  event failed.")
			} else if ne.eventType == EVENT_TYPE_DELETE {

				go utils.RunSafe(func() error {
					klog.Infof("delete ns %s to storage", ne.NamespaceName)
					return srv.deleteNsResource(ctx, ne)
				}, "sync namespace delete  event failed.")
			}
		case <-time.After(time.Minute * 5):
			klog.Info("No resource synchronization event occurred in the last 5 minutes...")
		}
	}
}

type podEvent struct {
	*entity.Pod
	eventType string
}

type nsEvent struct {
	*entity.Namespace
	eventType string
}

type kubeHandler struct {
	clusterUniqKey string // 区别事件所属集群
	clusterID      uint
	resourceKind   string // 区别资源类型
	podsChan       chan *podEvent
	nsChan         chan *nsEvent
}

func (kh *kubeHandler) sendEvent(obj interface{}, eventType string) {
	if pod, ok := obj.(*corev1.Pod); ok {
		if pod.Spec.InitContainers != nil {

		}

		kh.podsChan <- &podEvent{
			Pod: &entity.Pod{
				PodName:    pod.Name,
				Namespace:  pod.Namespace,
				ClusterRef: kh.clusterID,
				//ClusterUniqKey: kh.clusterUniqKey,
				ResourceKind: kh.resourceKind,
				Status:       pods.PodStatus(pod),
				PodIP:        pod.Status.PodIP,
			},
			eventType: eventType,
		}
	}

	if namespace, ok := obj.(*corev1.Namespace); ok {
		kh.nsChan <- &nsEvent{
			Namespace: &entity.Namespace{
				NamespaceName:  namespace.Name,
				ClusterUniqKey: kh.clusterUniqKey,
				ResourceKind:   kh.resourceKind,
			},
			eventType: eventType,
		}
	}
}

func (kh *kubeHandler) OnAdd(obj interface{}) {
	kh.sendEvent(obj, EVENT_TYPE_ADD)
}

func (kh *kubeHandler) OnUpdate(oldObj, newObj interface{}) {
	kh.sendEvent(newObj, EVENT_TYPE_UPDATE)
}

func (kh *kubeHandler) OnDelete(obj interface{}) {
	kh.sendEvent(obj, EVENT_TYPE_DELETE)
}

package service

import (
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"sync"
)

const (
	EVENT_TYPE_ADD    = "ADD"
	EVENT_TYPE_UPDATE = "UPDATE"
	EVENT_TYPE_DELETE = "DELETE"
)

type kubeHandlerServices struct {
	handlers     sync.Map
	podEventChan chan *podEvent
	nsEventChan  chan *nsEvent
}

func (srv *kubeHandlerServices) newHandler(kind, uk string) *kubeHandler {
	handler := &kubeHandler{
		clusterUniqKey: uk,
		resourceKind:   kind,
		podsChan:       srv.podEventChan,
		nsChan:         srv.nsEventChan,
	}
	srv.handlers.Store(srv.buildKey(handler.resourceKind, handler.clusterUniqKey), handler)
	return handler
}

func (srv *kubeHandlerServices) buildKey(kind, uk string) string {
	return utils.StringBuild(uk, "_", kind)
}

func (srv *kubeHandlerServices) loadHandler(kind, uk string) *kubeHandler {
	var handler *kubeHandler
	srv.handlers.Range(func(key, value any) bool {
		if key == srv.buildKey(kind, uk) {
			handler = value.(*kubeHandler)
			return true
		}
		return false
	})
	return handler
}

func (srv *kubeHandlerServices) delHandler(kind, uk string) {
	srv.handlers.Delete(srv.buildKey(kind, uk))
}

func (srv *kubeHandlerServices) handlerLoop() {
	for {
		select {
		case pe := <-srv.podEventChan:
			fmt.Println(pe.PodName)
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
	resourceKind   string // 区别资源类型
	podsChan       chan *podEvent
	nsChan         chan *nsEvent
}

func (kh *kubeHandler) sendEvent(obj interface{}, eventType string) {
	if pod, ok := obj.(*v1.Pod); ok {
		//pod.OwnerReferences
		kh.podsChan <- &podEvent{
			Pod: &entity.Pod{
				PodName:        pod.Name,
				ClusterUniqKey: kh.clusterUniqKey,
				ResourceKind:   kh.resourceKind,
			},
			eventType: eventType,
		}
	}

	if namespace, ok := obj.(*v1.Namespace); ok {
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

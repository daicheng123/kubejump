package service

import (
	"context"
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/kubernetes/pods"
	"github.com/toolkits/pkg/concurrent/semaphore"
	"github.com/toolkits/pkg/container/list"
	"github.com/toolkits/pkg/retry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

const (
	EVENT_TYPE_ADD           = "ADD"
	EVENT_TYPE_UPDATE        = "UPDATE"
	EVENT_TYPE_DELETE        = "DELETE"
	DEFAULT_RETRIES          = 3
	DEFAULT_RETRIES_INTERVAL = time.Millisecond * 500
)

// kubeHandlerServices  handler kubernetes event
type kubeHandlerServices struct {
	handlers         sync.Map
	podRepo          entity.PodRepo
	nsRepo           entity.NamespaceRepo
	eventConcurrency int
	eventQueue       *list.SafeListLimited
}

func (srv *kubeHandlerServices) newHandler(kind string, uniqKey string) *kubeHandler {
	handler := &kubeHandler{
		clusterUniqKey:      uniqKey,
		resourceKind:        kind,
		kubeHandlerServices: srv,
	}
	srv.handlers.Store(srv.buildKey(handler.resourceKind, handler.clusterUniqKey), handler)
	return handler
}

func (srv *kubeHandlerServices) buildKey(kind string, uniqKey string) string {
	return fmt.Sprintf("%s_%d", kind, uniqKey)
}

func (srv *kubeHandlerServices) loadHandler(kind string, uniqKey string) *kubeHandler {
	var handler *kubeHandler
	srv.handlers.Range(func(key, value any) bool {
		if key == srv.buildKey(kind, uniqKey) {
			handler = value.(*kubeHandler)
			return true
		}
		return false
	})
	return handler
}

func (srv *kubeHandlerServices) delHandler(kind string, clusterID string) {
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

func (srv *kubeHandlerServices) LoopHandler(ctx context.Context) {
	sema := semaphore.NewSemaphore(srv.eventConcurrency)
	duration := time.Duration(100) * time.Millisecond

	for {
		events := srv.eventQueue.PopBackBy(500)
		if len(events) == 0 {
			time.Sleep(duration)
			continue
		}
		srv.loopHandler(ctx, events, sema)
	}
}

func (srv *kubeHandlerServices) loopHandler(ctx context.Context, events []interface{}, sema *semaphore.Semaphore) {
	for i, event := range events {
		if events[i] == nil {
			continue
		}
		sema.Acquire()
		go func(event interface{}) {
			defer sema.Release()
			switch event.(type) {
			case *podEvent:
				srv.handlePod(ctx, event.(*podEvent))
			case *nsEvent:
				srv.handleNamespace(ctx, event.(*nsEvent))
			default:
				klog.Errorf("")
			}
		}(event)
	}
}

func (srv *kubeHandlerServices) handlePod(ctx context.Context, pe *podEvent) {
	var err error
	if pe.eventType != EVENT_TYPE_DELETE {
		if err = retry.Retry(DEFAULT_RETRIES, DEFAULT_RETRIES_INTERVAL, func() error {
			return srv.applyPodResource(ctx, pe)
		}); err != nil {
			klog.Errorf("sync pod add or update event failed, err: %s", err.Error())
		}
		return
	}

	if err = retry.Retry(DEFAULT_RETRIES, DEFAULT_RETRIES_INTERVAL, func() error {
		return srv.deletePodResource(ctx, pe)
	}); err != nil {
		klog.Errorf("sync pod delete event failed, err: %s", err.Error())
	}
}

func (srv *kubeHandlerServices) handleNamespace(ctx context.Context, ne *nsEvent) {
	var err error
	if ne.eventType != EVENT_TYPE_DELETE {
		if err = retry.Retry(DEFAULT_RETRIES, DEFAULT_RETRIES_INTERVAL, func() error {
			return srv.applyNsResource(ctx, ne)
		}); err != nil {
			klog.Errorf("sync namespace add or update event failed, err: %s", err.Error())
		}
		return
	}

	if err = retry.Retry(DEFAULT_RETRIES, DEFAULT_RETRIES_INTERVAL, func() error {
		return srv.deleteNsResource(ctx, ne)
	}); err != nil {
		klog.Errorf("sync pod delete event failed, err: %s", err.Error())
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
	clusterName    string
	resourceKind   string // 区别资源类型
	*kubeHandlerServices
}

func (kh *kubeHandler) sendEvent(obj interface{}, eventType string) {
	if pod, ok := obj.(*corev1.Pod); ok {
		if pod.Spec.InitContainers != nil {

		}
		kh.eventQueue.PushFront(&podEvent{
			Pod: &entity.Pod{
				PodName:    pod.Name,
				Namespace:  pod.Namespace,
				ClusterRef: kh.clusterUniqKey,
				//ClusterUniqKey: kh.clusterUniqKey,
				ResourceKind: kh.resourceKind,
				Status:       pods.PodStatus(pod),
				PodIP:        pod.Status.PodIP,
			},
			eventType: eventType,
		})
	}

	if namespace, ok := obj.(*corev1.Namespace); ok {
		kh.eventQueue.PushFront(&nsEvent{
			Namespace: &entity.Namespace{
				NamespaceName:  namespace.Name,
				ClusterUniqKey: kh.clusterUniqKey,
				ResourceKind:   kh.resourceKind,
			},
			eventType: eventType,
		})
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

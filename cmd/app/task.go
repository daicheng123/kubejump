package app

import (
	"context"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/kubernetes"
	"k8s.io/klog/v2"
)

// syncClusterResourcesToStore 同步各个已配置集群信息
func syncClusterResourcesToStore(server *server) {
	clusterConfigs, err := server.jmsService.ListClusterConfig(context.Background())
	if err != nil {
		klog.Errorf("[sync resource] query k8s cluster configs failed, err:[%s]", err.Error())
	}

	for _, cfg := range clusterConfigs {
		func(cfg *entity.ClusterConfig) {
			if err := server.k8sService.AddSyncResourceToStore(kubernetes.POD_INFORMER_NAME, cfg); err != nil {
				klog.Errorf("[sync resource] cluster %s add pod sync task failed, err:[%s]", cfg.ClusterName, err.Error())
			}
			if err := server.k8sService.AddSyncResourceToStore(kubernetes.NAMESPACE_INFORMER_NAME, cfg); err != nil {
				klog.Errorf("[sync resource] cluster %s add namespace sync task failed, err:[%s]", cfg.ClusterName, err.Error())
			}
		}(cfg)
	}
}

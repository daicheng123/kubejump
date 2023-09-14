package utils

import (
	"github.com/daicheng123/kubejump/internal/entity"
)

func PodsToJumpAssets(podInfo []*entity.Pod) []*entity.Asset {
	var assets = make([]*entity.Asset, 0, len(podInfo))
	for _, pod := range podInfo {
		assets = append(assets, &entity.Asset{
			ID:          pod.ID,
			Namespace:   pod.Namespace,
			PodIP:       pod.PodIP,
			PodName:     pod.PodName,
			ClusterName: pod.Cluster.ClusterName,
			PodStatus:   pod.Status,
			Cluster:     pod.Cluster,
		})
	}
	return assets
}

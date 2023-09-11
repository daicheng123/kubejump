package entity

import "sort"

type Asset struct {
	ID          uint
	ClusterName string
	Namespace   string
	PodName     string
	PodIP       string
	PodStatus   string
}

type assetSortBy func(asset1, asset2 *Asset) bool

func (by assetSortBy) Sort(nodes []*Asset) {
	nodeSorter := &AssetSorter{
		assets: nodes,
		sortBy: by,
	}

	sort.Sort(nodeSorter)
}

type AssetSorter struct {
	assets []*Asset
	sortBy func(node1, node2 *Asset) bool
}

func (a *AssetSorter) Len() int {
	return len(a.assets)
}

func (a *AssetSorter) Swap(i, j int) {
	a.assets[i], a.assets[j] = a.assets[j], a.assets[i]
}

func (a *AssetSorter) Less(i, j int) bool {
	return a.sortBy(a.assets[i], a.assets[j])
}

func clusterSort(asset1, asset2 *Asset) bool {
	return asset1.ClusterName < asset2.ClusterName
}

func sortIPSort(asset1, asset2 *Asset) bool {
	return asset1.PodIP < asset2.PodIP
}

func SortByClusterName(assets []*Asset) {
	assetSortBy(clusterSort).Sort(assets)
}

func SortByAssetIP(assets []*Asset) {
	assetSortBy(sortIPSort).Sort(assets)
}

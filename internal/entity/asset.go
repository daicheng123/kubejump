package entity

import (
	"fmt"
	"sort"
	"strings"
)

type Asset struct {
	ID uint
	//UniqKey     string
	ClusterName string
	Namespace   string
	PodName     string
	PodIP       string
	PodStatus   string
	Cluster     *ClusterConfig
}

func (a *Asset) String() string {
	return fmt.Sprintf("%s(%s)", a.PodName, a.PodIP)
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

type Domain struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Gateways []Gateway `json:"gateways"`
}

type Gateway struct {
	ID        string    `json:"id"`
	Name      string    `json:"Name"`
	Address   string    `json:"address"`
	Protocols Protocols `json:"protocols"`
	//Account   Account   `json:"account"`
}

type Protocols []Protocol

func (p Protocols) GetProtocolPort(protocol string) int {
	for i := range p {
		if strings.EqualFold(p[i].Name, protocol) {
			return p[i].Port
		}
	}
	return 0
}
func (p Protocols) IsSupportProtocol(protocol string) bool {
	for _, item := range p {
		protocolName := strings.ToLower(item.Name)
		if protocolName == strings.ToLower(protocol) {
			return true
		}
	}
	return false
}

type Protocol struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Port   int    `json:"port"`
	Public bool   `json:"public"`
}

const (
	ProtocolSSH    = "ssh"
	ProtocolTelnet = "telnet"
	ProtocolK8S    = "k8s"
	ProtocolSFTP   = "sftp"
	ProtocolRedis  = "redis"
)

package handler

import (
	"context"
	"fmt"
	"github.com/daicheng123/kubejump/config"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/common"
	"github.com/daicheng123/kubejump/pkg/utils"
	"k8s.io/klog/v2"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type dataSource string

const (
	loadingFromLocal  dataSource = "local"
	loadingFromRemote dataSource = "remote"
)

type selectType int

const (
	TypeAsset selectType = iota + 1
	TypeNodeAsset
	TypeK8s
	TypeDatabase
)

type UserSelectHandler struct {
	user *entity.User
	h    *InteractiveHandler

	loadingPolicy dataSource
	currentType   selectType
	searchKey     string

	hasPre  bool
	hasNext bool

	allLocalData []*entity.Asset

	selectedPodAsset *entity.Asset
	currentResult    []*entity.Asset

	*pageInfo
}

func (u *UserSelectHandler) HasNext() bool {
	return u.hasNext
}

func (u *UserSelectHandler) MoveNextPage() {
	fmt.Println("MoveNextPageMoveNextPageMoveNextPageMoveNextPageMoveNextPage")
	if u.HasNext() {
		offset := u.CurrentOffSet()
		newPageSize := getPageSize(u.h.term, config.GetConf().TerminalConf)
		u.currentResult = u.Retrieve(newPageSize, offset, u.searchKey)
	}
	u.DisplayCurrentResult()
}

func (u *UserSelectHandler) SetSelectPrepare() {
	u.SetLoadPolicy(loadingFromRemote) // default remote
	//u.AutoCompletion()
	u.h.term.SetPrompt("[Pods]> ")
	//u.currentType = s
}

func (u *UserSelectHandler) SetLoadPolicy(policy dataSource) {
	u.loadingPolicy = policy
}

func (u *UserSelectHandler) SetAllLocalAssetData(assets []*entity.Asset) {
	//u.allLocalData = make([]*entity.Asset, len(assets))

	u.currentResult = make([]*entity.Asset, len(assets))

	copy(u.allLocalData, assets)
}

func (u *UserSelectHandler) AutoCompletion() {
	assets := u.Retrieve(0, 0, "")
	suggests := make([]string, 0, len(assets))
	klog.Infof("[AutoCompletion] currentType %d", u.currentType)
	for _, a := range assets {
		suggests = append(suggests, a.PodName)
		//switch u.currentType {
		//case TypeAsset, TypeNodeAsset:
		//	suggests = append(suggests, a.PodName)
		//default:
		//	//suggests = append(suggests, a.ClusterName)
		//	suggests = append(suggests, a.PodName)
		//}
	}
	sort.Strings(suggests)
	klog.Infof("[AutoCompletion] suggestions: %#v", suggests)
	u.h.term.AutoCompleteCallback = func(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
		//if key == 9 {
		klog.Infof("line %s, pos %d,key %d, currenType: %d\n", line, pos, key, u.currentType)
		termWidth, _ := u.h.term.GetSize()
		if len(line) >= 1 {
			sugs := utils.FilterPrefix(suggests, line)
			if len(sugs) >= 1 {
				commonPrefix := utils.LongestCommonPrefix(sugs)
				fmt.Fprintf(u.h.term, "%s%s\n%s\n", "[Pods]> ", line, utils.Pretty(sugs, termWidth))
				//switch u.currentType {
				//case TypeAsset, TypeNodeAsset:
				//	fmt.Fprintf(u.h.term, "%s%s\n%s\n", "[Pods]> ", line, utils.Pretty(sugs, termWidth))
				//
				//}
				return commonPrefix, len(commonPrefix), true
			}
		}
		//}
		return newLine, newPos, false
	}
}

func (u *UserSelectHandler) Retrieve(pageSize, offset int, search string) []*entity.Asset {
	return u.retrieveFromRemote(pageSize, offset, search)

}

func (u *UserSelectHandler) retrieveFromRemote(pageSize, offset int, search string) []*entity.Asset {
	var order string

	order = "cluster_ref desc"
	reqParam := &entity.PaginationParam{
		PageSize: pageSize,
		Offset:   offset,
		Search:   search,
		SortBy:   order,
		IsActive: true,
	}
	resp, err := u.h.jmsService.ListPodsFromStorage(context.Background(), reqParam)

	if err != nil {
		klog.Errorf("Get user perm assets failed: %s", err.Error())
	}

	return u.updateRemotePageData(reqParam, resp)
	//
}

func (u *UserSelectHandler) updateRemotePageData(reqParam *entity.PaginationParam, res *entity.PaginationResponse) []*entity.Asset {

	u.hasNext, u.hasPre = false, false
	total := res.Total
	currentPageSize := reqParam.PageSize
	currentData := res.Data

	if currentPageSize < 0 || currentPageSize == PAGESIZEALL {
		currentPageSize = len(res.Data)
	}

	if len(res.Data) > currentPageSize {
		currentData = currentData[:currentPageSize]
	}
	currentOffset := reqParam.Offset + len(currentData)

	u.updatePageInfo(currentPageSize, total, currentOffset)

	if u.currentPage > 1 {
		u.hasPre = true
	}
	if u.currentPage < u.totalPage {
		u.hasNext = true
	}
	fmt.Printf("[updateRemotePageData] hasNext:%t, hasPre:%t,currentOffset:%d,currentPageSize:%d\n",
		u.hasNext, u.hasPre, currentOffset, currentPageSize)
	return currentData
}

func (u *UserSelectHandler) retrieveFromLocal(pageSize, offset int, search string) []*entity.Asset {
	if pageSize <= 0 {
		pageSize = PAGESIZEALL
	}
	if offset < 0 {
		offset = 0
	}
	searchResult := u.searchLocalAsset(search)
	var (
		totalData       []*entity.Asset
		total           int
		currentOffset   int
		currentPageSize int
	)

	if offset < len(searchResult) {
		totalData = searchResult[offset:]
	}
	total = len(totalData)
	currentPageSize = pageSize
	currentData := totalData

	if currentPageSize < 0 || currentPageSize == PAGESIZEALL {
		currentPageSize = len(totalData)
	}
	if total > currentPageSize {
		currentData = totalData[:currentPageSize]
	}
	currentOffset = offset + len(currentData)
	u.updatePageInfo(currentPageSize, total, currentOffset)
	u.hasPre = false
	u.hasNext = false
	if u.currentPage > 1 {
		u.hasPre = true
	}
	if u.currentPage < u.totalPage {
		u.hasNext = true
	}
	return currentData
}

func (u *UserSelectHandler) retrieveLocal(search string) []*entity.Asset {
	klog.Info("Retrieve default local data type: Asset")

	return u.searchLocalAsset(search)
	//switch u.currentType {
	//case TypeDatabase:
	//	return u.searchLocalDatabase(searches...)
	//case TypeK8s:
	//	return u.searchLocalK8s(searches...)
	//case TypeAsset:

	//default:
	//	// TypeAsset
	//	u.SetSelectType(TypeAsset)
	//
	//	return u.searchLocalAsset(searches...)
	//}
}

func (u *UserSelectHandler) searchLocalAsset(search string) []*entity.Asset {
	fields := map[string]struct{}{
		"ClusterName": {},
		"Namespace":   {},
		"PodName":     {},
		"PodIp":       {},
	}
	return u.searchLocalFromFields(fields, search)
}

func (u *UserSelectHandler) searchLocalFromFields(fields map[string]struct{}, search string) []*entity.Asset {
	items := make([]*entity.Asset, 0, len(u.allLocalData))
	for i := range u.allLocalData {
		if containKeysInMapItemFields(u.allLocalData[i], fields, search) {
			items = append(items, u.allLocalData[i])
		}
	}
	return items
}

func containKeysInMapItemFields(item *entity.Asset, searchFields map[string]struct{}, matchedKey string) bool {

	if len(matchedKey) == 0 {
		return true
	}
	//if len(matchedKey) == 1 && matchedKeys[0] == "" {
	//	return true
	//}

	v := reflect.ValueOf(item)
	for i := 0; i < v.NumField(); i++ {
		for key, _ := range searchFields {
			field := v.Type().Field(i)
			if field.Name == key {
				switch field.Type.Kind() {
				case reflect.String:
					if strings.Contains(v.Field(i).String(), matchedKey) {
						return true
					}
				}
			}

		}
	}
	return false
}

func (u *UserSelectHandler) Search(key string) {
	newPageSize := getPageSize(u.h.term, config.GetConf().TerminalConf)
	u.currentResult = u.Retrieve(newPageSize, 0, key)
	u.searchKey = key
	u.DisplayCurrentResult()
}

func (u *UserSelectHandler) DisplayCurrentResult() {
	searchHeader := fmt.Sprintf("Search: %s", u.searchKey)

	//switch u.currentType {
	//case TypeAsset:
	//	u.displayAssetResult(searchHeader)
	//default:
	//	klog.Error("Display unknown type")
	//}
	u.displayAssetResult(searchHeader)
}

func (u *UserSelectHandler) displayAssetResult(searchHeader string) {
	term := u.h.term
	if len(u.currentResult) == 0 {
		noAssets := "No Pod Assets"
		utils.IgnoreErrWriteString(term, utils.WrapperString(noAssets, utils.Red))
		utils.IgnoreErrWriteString(term, utils.CharNewLine)
		utils.IgnoreErrWriteString(term, utils.WrapperString(searchHeader, utils.Green))
		utils.IgnoreErrWriteString(term, utils.CharNewLine)
		return
	}
	u.displaySortedAssets(searchHeader)
}

func (u *UserSelectHandler) SearchAgain(key string) {
	u.searchKey = key
	newPageSize := getPageSize(u.h.term, u.h.terminalConf)
	u.currentResult = u.Retrieve(newPageSize, 0, u.searchKey)
	u.DisplayCurrentResult()
}

func (u *UserSelectHandler) displaySortedAssets(searchHeader string) {
	assetListSortBy := config.GetConf().TerminalConf.AssetListSortBy
	switch assetListSortBy {
	case "ip":
		entity.SortByAssetIP(u.currentResult)
	default:
		entity.SortByClusterName(u.currentResult)
	}
	term := u.h.term
	currentPage := u.CurrentPage()
	pageSize := u.PageSize()
	totalPage := u.TotalPage()
	totalCount := u.TotalCount()

	idLabel := "ID"
	clusterLabel := "ClusterName"
	nsLabel := "Namespace"
	podName := "PodName"
	podIPLabel := "PodIP"

	Labels := []string{idLabel, clusterLabel, nsLabel, podName, podIPLabel}
	fields := []string{"ID", "ClusterName", "Namespace", "PodName", "PodIP"}

	data := make([]map[string]string, len(u.currentResult))
	for i, j := range u.currentResult {
		row := make(map[string]string)
		row["ID"] = strconv.Itoa(i + 1)
		fieldMap := map[string]string{
			"ClusterName": "ClusterName",
			"Namespace":   "Namespace",
			"PodName":     "PodName",
			"PodIP":       "PodIP",
		}
		row = convertAssetItemToRow(j, fieldMap, row)
		data[i] = row
	}
	w, _ := term.GetSize()
	caption := fmt.Sprintf("Page: %d, Count: %d, Total Page: %d, Total Count: %d",
		currentPage, pageSize, totalPage, totalCount)

	caption = utils.WrapperString(caption, utils.Green)
	table := common.WrapperTable{
		Fields: fields,
		Labels: Labels,
		FieldsSize: map[string][3]int{
			"ID":           {0, 0, 5},
			"cluster_name": {0, 40, 0},
			"namespace":    {0, 15, 40},
			"pod_name":     {0, 0, 0},
			"pod_ip":       {0, 0, 0},
		},
		Data:        data,
		TotalSize:   w,
		Caption:     caption,
		TruncPolicy: common.TruncMiddle,
	}
	table.Initial()
	loginTip := "Enter ID number directly login the asset, multiple search use // + field, such as: //16"
	pageActionTip := "Page up: b Page down: n"
	actionTip := fmt.Sprintf("%s %s", loginTip, pageActionTip)

	_, _ = term.Write([]byte(utils.CharClear))
	_, _ = term.Write([]byte(table.Display()))
	utils.IgnoreErrWriteString(term, utils.WrapperString(actionTip, utils.Green))
	utils.IgnoreErrWriteString(term, utils.CharNewLine)
	utils.IgnoreErrWriteString(term, utils.WrapperString(searchHeader, utils.Green))
	utils.IgnoreErrWriteString(term, utils.CharNewLine)
}

func convertAssetItemToRow(item *entity.Asset, fields map[string]string, row map[string]string) map[string]string {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		for k, _ := range fields {
			field := v.Type().Field(i)
			if field.Name == k {
				switch field.Type.Kind() {
				case reflect.String:
					row[k] = v.Field(i).String()
				case reflect.Uint:
					row[k] = strconv.Itoa(int(v.Field(i).Uint()))
					continue
				}
			}
		}
	}
	return row
}

func (u *UserSelectHandler) SearchOrProxy(line string) {
	if indexNum, err := strconv.Atoi(line); err == nil && len(u.currentResult) > 0 {
		if indexNum > 0 && indexNum <= len(u.currentResult) {
			u.Proxy(u.currentResult[indexNum-1])
			return
		}
	}
}

func (u *UserSelectHandler) HasPrev() bool {
	return u.hasPre
}

func (u *UserSelectHandler) Proxy(target *entity.Asset) {
	//targetId := target.ID
	if target.PodStatus != "Running" {
		msg := "The pod is inactive"
		_, _ = u.h.term.Write([]byte(msg))
		return
	}
	u.proxyAsset(target)

	//lang := i18n.NewLang(u.h.i18nLang)
	//switch u.currentType {
	//case TypeAsset, TypeNodeAsset:
	//	asset, err := u.h.jmsService.GetAssetById(targetId)
	//	if err != nil || asset.ID == 0 {
	//		klog.Errorf("Select asset %s not found", targetId)
	//		return
	//	}
	//	if !asset.IsActive {
	//		logger.Debugf("Select asset %s is inactive", targetId)
	//		msg := lang.T("The asset is inactive")
	//		_, _ = u.h.term.Write([]byte(msg))
	//		return
	//	}
	//	u.proxyAsset(asset)
	//case TypeK8s, TypeDatabase:
	//	app, err := u.h.jmsService.GetApplicationById(targetId)
	//	if err != nil {
	//		logger.Errorf("Select application %s err: %s", targetId, err)
	//		return
	//	}
	//	u.proxyApp(app)
	//default:
	//	logger.Errorf("Select unknown type for target id %s", targetId)
	//}
}

func (u *UserSelectHandler) proxyAsset(asset *entity.Asset) {
	u.selectedPodAsset = asset

}

package handler

import (
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
	searchKeys    []string

	hasPre  bool
	hasNext bool

	allLocalData []entity.Asset

	//selectedNode  model.Node
	currentResult []entity.Asset

	*pageInfo
}

func (u *UserSelectHandler) HasNext() bool {
	return u.hasNext
}

func (u *UserSelectHandler) MoveNextPage() {
	if u.HasNext() {
		offset := u.CurrentOffSet()
		newPageSize := getPageSize(u.h.term, config.GetConf().TerminalConf)
		fmt.Printf("offset: %d, newPageSize: %d, \n", offset, newPageSize)
		//u.currentResult = u.Retrieve(newPageSize, offset, u.searchKeys...)
	}
	//u.DisplayCurrentResult()
}

func (u *UserSelectHandler) SetSelectType(s selectType) {
	u.SetLoadPolicy(loadingFromRemote) // default remote
	switch s {
	case TypeAsset:
		switch u.h.assetLoadPolicy {
		case "all":
			u.SetLoadPolicy(loadingFromLocal)
			u.AutoCompletion()
		}
		u.h.term.SetPrompt("[Pods]> ")
	}
	u.currentType = s
}

func (u *UserSelectHandler) SetLoadPolicy(policy dataSource) {
	u.loadingPolicy = policy
}

func (u *UserSelectHandler) SetAllLocalAssetData(assets []entity.Asset) {
	u.allLocalData = make([]entity.Asset, len(assets))
	copy(u.allLocalData, assets)
}

func (u *UserSelectHandler) AutoCompletion() {
	assets := u.Retrieve(0, 0, "")
	suggests := make([]string, 0, len(assets))
	klog.Infof("[AutoCompletion] currentType %d", u.currentType)
	for _, a := range assets {
		switch u.currentType {
		case TypeAsset, TypeNodeAsset:
			suggests = append(suggests, a.PodName)
		default:
			//suggests = append(suggests, a.ClusterName)
			suggests = append(suggests, a.PodName)
		}
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
				switch u.currentType {
				case TypeAsset, TypeNodeAsset:
					fmt.Fprintf(u.h.term, "%s%s\n%s\n", "[Pods]> ", line, utils.Pretty(sugs, termWidth))

				}
				return commonPrefix, len(commonPrefix), true
			}
		}
		//}
		return newLine, newPos, false
	}
}

// TODO: retrieveFromRemote
func (u *UserSelectHandler) Retrieve(pageSize, offset int, searches ...string) []entity.Asset {
	switch u.loadingPolicy {
	case loadingFromLocal:
		return u.retrieveFromLocal(pageSize, offset, searches...)
	default:
		return u.retrieveFromLocal(pageSize, offset, searches...)
	}
}

func (u *UserSelectHandler) retrieveFromLocal(pageSize, offset int, searches ...string) []entity.Asset {
	if pageSize <= 0 {
		pageSize = PAGESIZEALL
	}
	if offset < 0 {
		offset = 0
	}
	searchResult := u.searchLocalAsset(searches...)
	var (
		totalData       []entity.Asset
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

func (u *UserSelectHandler) retrieveLocal(searches ...string) []entity.Asset {
	klog.Info("Retrieve default local data type: Asset")

	return u.searchLocalAsset(searches...)
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

func (u *UserSelectHandler) searchLocalAsset(searches ...string) []entity.Asset {
	fields := map[string]struct{}{
		"ClusterName": {},
		"Namespace":   {},
		"PodName":     {},
		"PodIp":       {},
	}
	return u.searchLocalFromFields(fields, searches...)
}

func (u *UserSelectHandler) searchLocalFromFields(fields map[string]struct{}, searches ...string) []entity.Asset {
	items := make([]entity.Asset, 0, len(u.allLocalData))
	for i := range u.allLocalData {
		if containKeysInMapItemFields(u.allLocalData[i], fields, searches...) {
			items = append(items, u.allLocalData[i])
		}
	}
	return items
}

func containKeysInMapItemFields(item entity.Asset, searchFields map[string]struct{}, matchedKeys ...string) bool {

	if len(matchedKeys) == 0 {
		return true
	}
	if len(matchedKeys) == 1 && matchedKeys[0] == "" {
		return true
	}

	v := reflect.ValueOf(item)
	for i := 0; i < v.NumField(); i++ {
		for key, _ := range searchFields {
			field := v.Type().Field(i)
			if field.Name == key {
				switch field.Type.Kind() {
				case reflect.String:
					for j, _ := range matchedKeys {
						if strings.Contains(v.Field(i).String(), matchedKeys[j]) {
							return true
						}
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
	u.searchKeys = []string{key}
	u.DisplayCurrentResult()
}

func (u *UserSelectHandler) DisplayCurrentResult() {
	searchHeader := fmt.Sprintf("Search: %s", strings.Join(u.searchKeys, " "))
	switch u.currentType {
	case TypeAsset:
		u.displayAssetResult(searchHeader)
	default:
		klog.Error("Display unknown type")
	}
}

func (u *UserSelectHandler) displayAssetResult(searchHeader string) {
	term := u.h.term
	if len(u.currentResult) == 0 {
		noAssets := "No Assets"
		utils.IgnoreErrWriteString(term, utils.WrapperString(noAssets, utils.Red))
		utils.IgnoreErrWriteString(term, utils.CharNewLine)
		utils.IgnoreErrWriteString(term, utils.WrapperString(searchHeader, utils.Green))
		utils.IgnoreErrWriteString(term, utils.CharNewLine)
		return
	}
	u.displaySortedAssets(searchHeader)
}

func (u *UserSelectHandler) displaySortedAssets(searchHeader string) {
	assetListSortBy := config.GetConf().TerminalConf.AssetListSortBy
	switch assetListSortBy {
	//case "ip":
	//	sortedAsset := IPAssetList(u.currentResult)
	//	sort.Sort(sortedAsset)
	//	u.currentResult = sortedAsset
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
			"ID":          {0, 0, 5},
			"ClusterName": {0, 40, 0},
			"Namespace":   {0, 15, 40},
			"PodName":     {0, 0, 0},
			"PodIP":       {0, 0, 0},
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

func convertAssetItemToRow(item entity.Asset, fields map[string]string, row map[string]string) map[string]string {
	v := reflect.ValueOf(item)
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

func (u *UserSelectHandler) Proxy(target entity.Asset) {
	//targetId := target.ID
	if target.PodStatus != "Running" {
		msg := "The pod is inactive"
		_, _ = u.h.term.Write([]byte(msg))
		return
	}
	//u.proxyAsset(asset)

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

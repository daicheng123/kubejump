package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
)

func CheckParams(c *gin.Context, ptr interface{}) error {
	if ptr == nil {
		return nil
	}
	switch t := ptr.(type) {
	case string:
		if t != "" {
			panic(t)
		}
	case error:
		panic(t.Error())
	}

	if err := c.ShouldBindJSON(&ptr); err != nil {
		klog.Warningf(fmt.Sprintf("解析参数出错：%v", err.Error()))
		return err
	}
	return nil
}

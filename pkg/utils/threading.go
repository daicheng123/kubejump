package utils

import (
	"fmt"
	"k8s.io/klog/v2"
)

//func RunSafe(fun func() (err error), errMsg string) {
//	defer Recover()
//
//	if err := fun(); err != nil {
//		klog.Error("[threading] %s: %s", errMsg, err.Error())
//	}
//}

func RunSafe(fn func() error, errMsg string) {
	defer func() {
		if p := recover(); p != nil {
			klog.Errorf("[threading] %s", p)
		}
	}()

	if err := fn(); err != nil {
		klog.Errorf(fmt.Sprintf("%s: %s", errMsg, err.Error()))
	}
}

//func RunSafeWithMsg(fn func() error, errTemplate string) {
//	RunSafe(fn, func(err interface{}) {
//		klog.Errorf(fmt.Sprintf("%s: %s", errTemplate, err))
//	})
//}
//
//
//
//func Recover(errHandler func(err interface{})) {
//	if p := recover(); p != nil {
//		errHandler(p)
//	}
//}

package httpd

import (
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/daicheng123/kubejump/pkg/utils"
	"github.com/gin-gonic/gin"
	"time"
)

func (s *Server) HealthCheck(ctx *gin.Context) {
	status := make(map[string]interface{})
	status["timestamp"] = time.Now().UTC()
	utils.OkWithData(status, ctx)
}

func (s *Server) ApplyK8sCluster(ctx *gin.Context) {
	clusterReq := new(entity.ClusterConfig)

	err := utils.CheckParams(ctx, clusterReq)
	if err != nil {
		utils.FailWithMessage(utils.ParamError, err.Error(), ctx)
		return
	}
	cluster, err := s.jmsService.ApplyK8sCluster(ctx, clusterReq)
	if err != nil {
		utils.FailWithMessage(utils.CreateK8SClusterError, err.Error(), ctx)
		return
	}
	utils.OkWithData(cluster, ctx)
}

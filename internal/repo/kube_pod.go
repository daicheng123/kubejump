package repo

import (
	"github.com/daicheng123/kubejump/internal/base/data"
	"github.com/daicheng123/kubejump/internal/entity"
)

type PodRepo struct {
	data *data.Data
}

func (pr *PodRepo) AddPod(pod *entity.Pod) error {

	return nil
}

package session

import (
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
)

type Session struct {
	*entity.Session
	handleTaskFunc func(task *entity.TerminalTask) error
}

func (s *Session) HandleTask(task *entity.TerminalTask) error {
	if s.handleTaskFunc != nil {
		return s.handleTaskFunc(task)
	}
	return fmt.Errorf("no handle task func")
}

func NewSession(s *entity.Session, handleTaskFunc func(task *entity.TerminalTask) error) *Session {
	return &Session{Session: s, handleTaskFunc: handleTaskFunc}
}

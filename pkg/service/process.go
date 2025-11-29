package service

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Service struct {
	Name        string
	Command     string
	Status      string
	PID         int
	CreatedAt   time.Time
	LastStarted time.Time
	LogFile     string
}

func (s *Service) Start() error {
	// 构建完整的命令，使用nohup和&，并使用 >> 追加日志
	cmdStr := fmt.Sprintf("nohup %s >> %s 2>&1 & echo $!", s.Command, s.LogFile)
	cmd := exec.Command("sh", "-c", cmdStr)

	// 执行命令并获取输出
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	// 解析PID
	pidStr := strings.TrimSpace(string(output))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("failed to parse PID: %v", err)
	}

	s.PID = pid
	s.LastStarted = time.Now()

	return nil
}

func (s *Service) Stop() error {
	if s.PID == 0 {
		return nil
	}

	// 检查进程是否存在
	if err := syscall.Kill(s.PID, 0); err != nil {
		// 进程不存在，直接清理
		s.PID = 0
		return nil
	}

	// 强制终止进程
	if err := syscall.Kill(s.PID, syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	s.PID = 0

	return nil
}

func (s *Service) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

func (s *Service) GetLogs() (string, error) {
	data, err := os.ReadFile(s.LogFile)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %v", err)
	}
	return string(data), nil
}

func (s *Service) IsRunning() bool {
	if s.PID == 0 {
		return false
	}

	// 使用 syscall.Kill(pid, 0) 检查进程是否存在
	// 如果返回 nil，说明进程存在且有权限发送信号
	// 如果返回 EPERM，说明进程存在但无权限（由于我们是管理自己的进程，通常意味着存在）
	// 如果返回 ESRCH，说明进程不存在
	err := syscall.Kill(s.PID, 0)
	return err == nil || err == syscall.EPERM
}

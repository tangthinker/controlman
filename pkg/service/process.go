package service

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (s *Service) Start() error {
	// 构建完整的命令，使用nohup和&
	cmdStr := fmt.Sprintf("nohup %s > %s 2>&1 & echo $!", s.Command, s.LogFile)
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
	s.Status = "running"
	s.LastStarted = time.Now()

	// 保存PID到文件
	if err := os.WriteFile(s.PIDFile, []byte(strconv.Itoa(s.PID)), 0644); err != nil {
		return fmt.Errorf("failed to save PID file: %v", err)
	}

	return nil
}

func (s *Service) Stop() error {
	if s.PID == 0 {
		return fmt.Errorf("service is not running")
	}

	// 强制终止进程
	cmd := exec.Command("kill", "-9", strconv.Itoa(s.PID))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	s.Status = "stopped"
	s.PID = 0

	// 删除PID文件
	if err := os.Remove(s.PIDFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %v", err)
	}

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

	// 检查进程是否存在
	cmd := exec.Command("ps", "-p", strconv.Itoa(s.PID))
	return cmd.Run() == nil
}

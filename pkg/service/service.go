package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Service struct {
	Name        string    `json:"name"`
	Command     string    `json:"command"`
	Status      string    `json:"status"` // running, stopped, failed
	PID         int       `json:"pid"`
	CreatedAt   time.Time `json:"created_at"`
	LastStarted time.Time `json:"last_started"`
	LogFile     string    `json:"log_file"`
	PIDFile     string    `json:"pid_file"`
}

type ServiceManager struct {
	BaseDir string
}

func NewServiceManager() (*ServiceManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(homeDir, ".controlman")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	return &ServiceManager{
		BaseDir: baseDir,
	}, nil
}

func (sm *ServiceManager) GetServiceDir(name string) string {
	return filepath.Join(sm.BaseDir, name)
}

func (sm *ServiceManager) GetServiceConfigPath(name string) string {
	return filepath.Join(sm.GetServiceDir(name), "config.json")
}

func (sm *ServiceManager) SaveService(s *Service) error {
	serviceDir := sm.GetServiceDir(s.Name)
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}

	s.LogFile = filepath.Join(serviceDir, "service.log")
	s.PIDFile = filepath.Join(serviceDir, "service.pid")

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sm.GetServiceConfigPath(s.Name), data, 0644)
}

func (sm *ServiceManager) LoadService(name string) (*Service, error) {
	data, err := os.ReadFile(sm.GetServiceConfigPath(name))
	if err != nil {
		return nil, err
	}

	var s Service
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (sm *ServiceManager) ListServices() ([]*Service, error) {
	entries, err := os.ReadDir(sm.BaseDir)
	if err != nil {
		return nil, err
	}

	var services []*Service
	for _, entry := range entries {
		if entry.IsDir() {
			service, err := sm.LoadService(entry.Name())
			if err != nil {
				continue
			}
			services = append(services, service)
		}
	}

	return services, nil
}

func (sm *ServiceManager) DeleteService(name string) error {
	serviceDir := sm.GetServiceDir(name)
	return os.RemoveAll(serviceDir)
}

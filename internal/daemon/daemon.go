package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/tangthinker/controlman/pkg/service"
)

type Daemon struct {
	serviceManager *service.ServiceManager
	socketPath     string
	services       map[string]*service.Service
	monitors       map[string]chan struct{} // 用于停止监控协程
}

type Command struct {
	Action  string          `json:"action"`
	Name    string          `json:"name"`
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func NewDaemon() (*Daemon, error) {
	// 创建 /var/run/controlman 目录
	if err := os.MkdirAll("/var/run/controlman", 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}

	socketPath := "/var/run/controlman/controlman.sock"
	serviceManager, err := service.NewServiceManager()
	if err != nil {
		return nil, err
	}

	d := &Daemon{
		serviceManager: serviceManager,
		socketPath:     socketPath,
		services:       make(map[string]*service.Service),
		monitors:       make(map[string]chan struct{}),
	}

	// 加载所有已存在的服务
	if err := d.loadServices(); err != nil {
		log.Printf("Warning: failed to load services: %v", err)
	}

	return d, nil
}

func (d *Daemon) loadServices() error {
	services, err := d.serviceManager.ListServices()
	if err != nil {
		return err
	}

	for _, s := range services {
		d.services[s.Name] = s
		// 启动服务
		if err := s.Start(); err != nil {
			log.Printf("Warning: failed to start service %s: %v", s.Name, err)
			continue
		}
		go d.monitorService(s)
	}

	return nil
}

func (d *Daemon) monitorService(s *service.Service) {
	stopChan := make(chan struct{})
	d.monitors[s.Name] = stopChan

	for {
		select {
		case <-stopChan:
			return
		default:
			if !s.IsRunning() {
				log.Printf("Service %s is not running, attempting to restart...", s.Name)
				if err := s.Restart(); err != nil {
					log.Printf("Failed to restart service %s: %v", s.Name, err)
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func (d *Daemon) handleCommand(cmd Command) Response {
	switch cmd.Action {
	case "add":
		return d.handleAdd(cmd)
	case "stop":
		return d.handleStop(cmd)
	case "start":
		return d.handleStart(cmd)
	case "logs":
		return d.handleLogs(cmd)
	case "list":
		return d.handleList()
	case "delete":
		return d.handleDelete(cmd)
	default:
		return Response{Success: false, Message: "unknown command"}
	}
}

func (d *Daemon) handleAdd(cmd Command) Response {
	if cmd.Name == "" || cmd.Command == "" {
		return Response{Success: false, Message: "name and command are required"}
	}

	if _, exists := d.services[cmd.Name]; exists {
		return Response{Success: false, Message: "service already exists"}
	}

	log.Printf("Adding new service: %s", cmd.Name)
	s := &service.Service{
		Name:      cmd.Name,
		Command:   cmd.Command,
		Status:    "stopped",
		CreatedAt: time.Now(),
	}

	if err := d.serviceManager.SaveService(s); err != nil {
		log.Printf("Failed to save service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to save service: %v", err)}
	}

	// 启动服务
	if err := s.Start(); err != nil {
		log.Printf("Failed to start service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to start service: %v", err)}
	}

	log.Printf("Service %s started successfully with PID %d", cmd.Name, s.PID)
	d.services[cmd.Name] = s
	go d.monitorService(s)

	return Response{Success: true, Message: "service added and started successfully"}
}

func (d *Daemon) handleStop(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, exists := d.services[cmd.Name]
	if !exists {
		return Response{Success: false, Message: "service not found"}
	}

	log.Printf("Stopping service: %s (PID: %d)", cmd.Name, s.PID)
	if err := s.Stop(); err != nil {
		log.Printf("Failed to stop service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to stop service: %v", err)}
	}

	// 停止监控协程
	if stopChan, exists := d.monitors[cmd.Name]; exists {
		close(stopChan)
		delete(d.monitors, cmd.Name)
	}

	log.Printf("Service %s stopped successfully", cmd.Name)
	return Response{Success: true, Message: "service stopped successfully"}
}

func (d *Daemon) handleStart(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, exists := d.services[cmd.Name]
	if !exists {
		return Response{Success: false, Message: "service not found"}
	}

	// 如果服务已经在运行，直接返回成功
	if s.IsRunning() {
		log.Printf("Service %s is already running (PID: %d)", cmd.Name, s.PID)
		return Response{Success: true, Message: "service is already running"}
	}

	log.Printf("Starting service: %s", cmd.Name)
	// 启动服务
	if err := s.Start(); err != nil {
		log.Printf("Failed to start service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to start service: %v", err)}
	}

	log.Printf("Service %s started successfully with PID %d", cmd.Name, s.PID)
	// 启动监控协程
	go d.monitorService(s)

	return Response{Success: true, Message: "service started successfully"}
}

func (d *Daemon) handleLogs(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, exists := d.services[cmd.Name]
	if !exists {
		return Response{Success: false, Message: "service not found"}
	}

	logs, err := s.GetLogs()
	if err != nil {
		return Response{Success: false, Message: fmt.Sprintf("failed to get logs: %v", err)}
	}

	return Response{Success: true, Data: logs}
}

func (d *Daemon) handleList() Response {
	services := make([]map[string]interface{}, 0)
	for _, s := range d.services {
		services = append(services, map[string]interface{}{
			"name":       s.Name,
			"status":     s.Status,
			"pid":        s.PID,
			"created_at": s.CreatedAt.Format(time.RFC3339),
			"last_start": s.LastStarted.Format(time.RFC3339),
			"command":    s.Command,
		})
	}
	return Response{Success: true, Data: services}
}

func (d *Daemon) handleDelete(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, exists := d.services[cmd.Name]
	if !exists {
		return Response{Success: false, Message: "service not found"}
	}

	log.Printf("Deleting service: %s (PID: %d)", cmd.Name, s.PID)
	if err := s.Stop(); err != nil {
		log.Printf("Warning: failed to stop service %s before deletion: %v", cmd.Name, err)
	}

	// 停止监控协程
	if stopChan, exists := d.monitors[cmd.Name]; exists {
		close(stopChan)
		delete(d.monitors, cmd.Name)
	}

	if err := d.serviceManager.DeleteService(cmd.Name); err != nil {
		log.Printf("Failed to delete service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to delete service: %v", err)}
	}

	delete(d.services, cmd.Name)
	log.Printf("Service %s deleted successfully", cmd.Name)
	return Response{Success: true, Message: "service deleted successfully"}
}

func (d *Daemon) Run() error {
	// 确保socket目录存在
	if err := os.MkdirAll(filepath.Dir(d.socketPath), 0755); err != nil {
		return err
	}

	// 删除已存在的socket文件
	os.Remove(d.socketPath)

	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("Daemon listening on %s", d.socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go d.handleConnection(conn)
	}
}

func (d *Daemon) handleConnection(conn net.Conn) {
	defer conn.Close()

	var cmd Command
	if err := json.NewDecoder(conn).Decode(&cmd); err != nil {
		json.NewEncoder(conn).Encode(Response{Success: false, Message: fmt.Sprintf("invalid command: %v", err)})
		return
	}

	response := d.handleCommand(cmd)
	json.NewEncoder(conn).Encode(response)
}

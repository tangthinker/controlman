package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tangthinker/controlman/pkg/service"
)

type Daemon struct {
	serviceManager *service.ServiceManager
	socketPath     string
	monitors       map[string]chan struct{} // 用于停止监控协程
	mu             sync.Mutex               // Protects monitors map
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(homeDir, ".controlman")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}

	socketPath := filepath.Join(baseDir, "controlman.sock")
	serviceManager, err := service.NewServiceManager()
	if err != nil {
		return nil, err
	}

	d := &Daemon{
		serviceManager: serviceManager,
		socketPath:     socketPath,
		monitors:       make(map[string]chan struct{}),
	}

	// Start log rotation
	d.StartLogRotationRoutine()

	// 加载所有已存在的服务
	if err := d.loadServices(); err != nil {
		log.Printf("Warning: failed to load services: %v", err)
	}

	return d, nil
}

func (d *Daemon) Close() error {
	d.mu.Lock()
	// Stop all monitors
	for name, ch := range d.monitors {
		close(ch)
		delete(d.monitors, name)
	}
	d.mu.Unlock()

	services, err := d.serviceManager.ListServices()
	if err != nil {
		return err
	}
	for _, s := range services {
		s.Stop()
		if err := d.serviceManager.SaveService(s); err != nil {
			log.Printf("Warning: failed to save service status %s: %v", s.Name, err)
		}
	}

	return d.serviceManager.Close()
}

func (d *Daemon) loadServices() error {
	services, err := d.serviceManager.ListServices()
	if err != nil {
		return err
	}

	for _, s := range services {
		// 启动服务
		if err := s.Start(); err != nil {
			// 如果 Start() 失败，process.go 内部会设置为 Failed，我们需要保存这个状态
			s.Status = service.StatusFailed
			if err := d.serviceManager.SaveService(s); err != nil {
				log.Printf("Warning: failed to update service status %s: %v", s.Name, err)
			}
			log.Printf("Warning: failed to start service %s: %v", s.Name, err)
			continue
		}
		// 更新服务状态
		s.Status = service.StatusRunning
		if err := d.serviceManager.SaveService(s); err != nil {
			log.Printf("Warning: failed to update service status %s: %v", s.Name, err)
		}
		go d.monitorService(s.Name)
	}

	return nil
}

func (d *Daemon) monitorService(name string) {
	stopChan := make(chan struct{})
	d.mu.Lock()
	d.monitors[name] = stopChan
	d.mu.Unlock()

	for {
		select {
		case <-stopChan:
			return
		default:
			s, err := d.serviceManager.LoadService(name)
			if err != nil {
				if err == os.ErrNotExist {
					log.Printf("Service %s no longer exists, stopping monitor", name)
					d.mu.Lock()
					if ch, ok := d.monitors[name]; ok && ch == stopChan {
						delete(d.monitors, name)
					}
					d.mu.Unlock()
					return
				}
				log.Printf("Failed to load service %s for monitoring: %v", name, err)
				time.Sleep(5 * time.Second)
				continue
			}

			if !s.IsRunning() {
				// 如果期望是运行中，但实际没运行，才需要重启
				// 注意：LoadService 得到的是最新状态，如果用户执行了 Stop，状态会变成 Stopped
				if s.Status == service.StatusRunning {
					log.Printf("Service %s is not running (expected Running), attempting to restart...", s.Name)

					// 设置为重启中
					s.Status = service.StatusRestarting
					d.serviceManager.SetServiceStatus(s.Name, service.StatusRestarting)

					if err := s.Restart(); err != nil {
						s.Status = service.StatusFailed
						d.serviceManager.SetServiceStatus(s.Name, service.StatusFailed)
						log.Printf("Failed to restart service %s: %v", s.Name, err)
					} else {
						// 重启成功，更新为 Running 并保存 PID 等信息
						s.Status = service.StatusRunning
						if err := d.serviceManager.SaveService(s); err != nil {
							log.Printf("Failed to save restarted service state %s: %v", s.Name, err)
						}
					}
				}
			}
			time.Sleep(1 * time.Second)
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
	case "restart":
		return d.handleRestart(cmd)
	case "logs":
		return d.handleLogs(cmd)
	case "info":
		return d.handleInfo(cmd)
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

	if _, err := d.serviceManager.LoadService(cmd.Name); err == nil {
		return Response{Success: false, Message: "service already exists"}
	}

	log.Printf("Adding new service: %s", cmd.Name)
	s := &service.Service{
		Name:      cmd.Name,
		Command:   cmd.Command,
		Status:    service.StatusStopped,
		CreatedAt: time.Now(),
	}

	if err := d.serviceManager.SaveService(s); err != nil {
		log.Printf("Failed to save service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to save service: %v", err)}
	}

	// 启动服务
	if err := s.Start(); err != nil {
		s.Status = service.StatusFailed
		d.serviceManager.SaveService(s) // 保存 Failed 状态
		log.Printf("Failed to start service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to start service: %v", err)}
	}

	// 保存启动后的状态
	s.Status = service.StatusRunning
	if err := d.serviceManager.SaveService(s); err != nil {
		log.Printf("Failed to save started service %s: %v", cmd.Name, err)
	}

	log.Printf("Service %s started successfully with PID %d", cmd.Name, s.PID)
	go d.monitorService(s.Name)

	return Response{Success: true, Message: "service added and started successfully"}
}

func (d *Daemon) handleStop(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, err := d.serviceManager.LoadService(cmd.Name)
	if err != nil {
		return Response{Success: false, Message: "service not found"}
	}

	log.Printf("Stopping service: %s (PID: %d)", cmd.Name, s.PID)
	// 先保存一个 Stopping 状态（可选，如果希望 UI 看到中间态）
	s.Status = service.StatusStopping
	if err := d.serviceManager.SetServiceStatus(s.Name, service.StatusStopping); err != nil {
		log.Printf("Failed to save stopping status for service %s: %v", s.Name, err)
	}

	if err := s.Stop(); err != nil {
		// 即使停止失败，也更新状态为之前保存的状态（Stopping），或者考虑设为 Running/Unknown
		// 这里如果不做任何操作，状态仍然是 Stopping。
		// 更好的做法可能是回滚为 Running 或 Failed
		s.Status = service.StatusRunning
		d.serviceManager.SetServiceStatus(s.Name, service.StatusRunning)
		log.Printf("Failed to stop service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to stop service: %v", err)}
	}

	// 保存停止后的状态
	s.Status = service.StatusStopped
	if err := d.serviceManager.SaveService(s); err != nil {
		log.Printf("Failed to save stopped service %s: %v", cmd.Name, err)
	}

	// 停止监控协程
	d.mu.Lock()
	if stopChan, exists := d.monitors[cmd.Name]; exists {
		close(stopChan)
		delete(d.monitors, cmd.Name)
	}
	d.mu.Unlock()

	log.Printf("Service %s stopped successfully", cmd.Name)
	return Response{Success: true, Message: "service stopped successfully"}
}

func (d *Daemon) handleStart(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, err := d.serviceManager.LoadService(cmd.Name)
	if err != nil {
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
		s.Status = service.StatusFailed
		d.serviceManager.SaveService(s) // 保存 Failed 状态
		log.Printf("Failed to start service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to start service: %v", err)}
	}

	// 保存启动后的状态
	s.Status = service.StatusRunning
	if err := d.serviceManager.SaveService(s); err != nil {
		log.Printf("Failed to save started service %s: %v", cmd.Name, err)
	}

	log.Printf("Service %s started successfully with PID %d", cmd.Name, s.PID)
	// 启动监控协程
	go d.monitorService(s.Name)

	return Response{Success: true, Message: "service started successfully"}
}

func (d *Daemon) handleRestart(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, err := d.serviceManager.LoadService(cmd.Name)
	if err != nil {
		return Response{Success: false, Message: "service not found"}
	}

	log.Printf("Restarting service: %s", cmd.Name)

	// Update status to restarting
	s.Status = service.StatusRestarting
	if err := d.serviceManager.SaveService(s); err != nil {
		log.Printf("Failed to save restarting status for service %s: %v", s.Name, err)
	}

	if err := s.Restart(); err != nil {
		s.Status = service.StatusFailed
		d.serviceManager.SaveService(s)
		log.Printf("Failed to restart service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to restart service: %v", err)}
	}

	s.Status = service.StatusRunning
	if err := d.serviceManager.SaveService(s); err != nil {
		log.Printf("Failed to save restarted service %s: %v", cmd.Name, err)
	}

	log.Printf("Service %s restarted successfully with PID %d", cmd.Name, s.PID)

	// Ensure monitor is running
	d.mu.Lock()
	_, exists := d.monitors[cmd.Name]
	d.mu.Unlock()

	if !exists {
		go d.monitorService(s.Name)
	}

	return Response{Success: true, Message: "service restarted successfully"}
}

func (d *Daemon) handleLogs(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, err := d.serviceManager.LoadService(cmd.Name)
	if err != nil {
		return Response{Success: false, Message: "service not found"}
	}

	logs, err := s.GetLogs()
	if err != nil {
		return Response{Success: false, Message: fmt.Sprintf("failed to get logs: %v", err)}
	}

	return Response{Success: true, Data: logs}
}

func (d *Daemon) handleList() Response {
	services, err := d.serviceManager.ListServices()
	if err != nil {
		return Response{Success: false, Message: fmt.Sprintf("failed to list services: %v", err)}
	}

	serviceList := make([]map[string]interface{}, 0)
	for _, s := range services {
		cpu, mem, _ := s.GetStats()
		serviceList = append(serviceList, map[string]interface{}{
			"name":       s.Name,
			"status":     s.Status,
			"pid":        s.PID,
			"cpu":        cpu,
			"memory":     mem,
			"created_at": s.CreatedAt.Format(time.RFC3339),
			"last_start": s.LastStarted.Format(time.RFC3339),
			"command":    s.Command,
		})
	}
	return Response{Success: true, Data: serviceList}
}

func (d *Daemon) handleDelete(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, err := d.serviceManager.LoadService(cmd.Name)
	if err != nil {
		return Response{Success: false, Message: "service not found"}
	}

	log.Printf("Deleting service: %s (PID: %d)", cmd.Name, s.PID)
	if err := s.Stop(); err != nil {
		log.Printf("Warning: failed to stop service %s before deletion: %v", cmd.Name, err)
	}

	// 停止监控协程
	d.mu.Lock()
	if stopChan, exists := d.monitors[cmd.Name]; exists {
		close(stopChan)
		delete(d.monitors, cmd.Name)
	}
	d.mu.Unlock()

	if err := d.serviceManager.DeleteService(cmd.Name); err != nil {
		log.Printf("Failed to delete service %s: %v", cmd.Name, err)
		return Response{Success: false, Message: fmt.Sprintf("failed to delete service: %v", err)}
	}

	log.Printf("Service %s deleted successfully", cmd.Name)
	return Response{Success: true, Message: "service deleted successfully"}
}

func (d *Daemon) handleInfo(cmd Command) Response {
	if cmd.Name == "" {
		return Response{Success: false, Message: "service name is required"}
	}

	s, err := d.serviceManager.LoadService(cmd.Name)
	if err != nil {
		return Response{Success: false, Message: "service not found"}
	}

	cpu, mem, _ := s.GetStats()

	info := map[string]interface{}{
		"name":       s.Name,
		"status":     s.Status,
		"pid":        s.PID,
		"cpu":        cpu,
		"memory":     mem,
		"created_at": s.CreatedAt.Format(time.RFC3339),
		"last_start": s.LastStarted.Format(time.RFC3339),
		"command":    s.Command,
		"log_file":   s.LogFile,
	}

	return Response{Success: true, Data: info}
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

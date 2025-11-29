package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
)

const (
	// DB Prefix
	prefixService = "services"
	separator     = ":"

	// Field names
	fieldCommand     = "command"
	fieldStatus      = "status"
	fieldPID         = "pid"
	fieldCreatedAt   = "created_at"
	fieldLastStarted = "last_started"
	fieldLogFile     = "log_file"

	StatusRunning    = "running"
	StatusStopped    = "stopped"
	StatusFailed     = "failed"
	StatusStarting   = "starting"
	StatusStopping   = "stopping"
	StatusRestarting = "restarting"
	StatusUnknown    = "unknown"
)

type ServiceManager struct {
	baseDir string
	db      *pebble.DB
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

	// Initialize Pebble DB
	dataDir := filepath.Join(baseDir, "data")
	db, err := pebble.Open(dataDir, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble db: %v", err)
	}

	return &ServiceManager{
		baseDir: baseDir,
		db:      db,
	}, nil
}

func (sm *ServiceManager) Close() error {
	if sm.db != nil {
		return sm.db.Close()
	}
	return nil
}

func (sm *ServiceManager) GetServiceDir(name string) string {
	return filepath.Join(sm.baseDir, name)
}

func (sm *ServiceManager) SetServiceStatus(name string, status string) error {
	key := makeKey(name, fieldStatus)
	return sm.db.Set(key, []byte(status), pebble.Sync)
}

func (sm *ServiceManager) SaveService(s *Service) error {
	// Ensure log directory exists
	serviceDir := sm.GetServiceDir(s.Name)
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}

	s.LogFile = filepath.Join(serviceDir, "service.log")

	batch := sm.db.NewBatch()
	defer batch.Close()

	if err := sm.saveServiceToBatch(batch, s); err != nil {
		return err
	}

	return batch.Commit(pebble.Sync)
}

func (sm *ServiceManager) saveServiceToBatch(batch *pebble.Batch, s *Service) error {
	k := func(field string) []byte {
		return makeKey(s.Name, field)
	}

	// Helper to handle error checking for batch.Set
	set := func(field, val string) error {
		return batch.Set(k(field), []byte(val), nil)
	}

	updates := map[string]string{
		fieldCommand:     s.Command,
		fieldStatus:      s.Status,
		fieldPID:         strconv.Itoa(s.PID),
		fieldCreatedAt:   s.CreatedAt.Format(time.RFC3339),
		fieldLastStarted: s.LastStarted.Format(time.RFC3339),
		fieldLogFile:     s.LogFile,
	}

	for field, val := range updates {
		if err := set(field, val); err != nil {
			return err
		}
	}
	return nil
}

func (sm *ServiceManager) LoadService(name string) (*Service, error) {
	prefix := makePrefix(name)
	upperBound := makeUpperBound(name)

	iter, err := sm.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	s := &Service{Name: name}
	found := false

	for iter.First(); iter.Valid(); iter.Next() {
		found = true
		parseField(s, string(iter.Key()), string(iter.Value()))
	}

	if !found {
		return nil, os.ErrNotExist
	}
	return s, nil
}

func (sm *ServiceManager) ListServices() ([]*Service, error) {
	prefix := []byte(prefixService + separator)
	upperBound := []byte(prefixService + ";")

	iter, err := sm.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var services []*Service
	var currentService *Service
	var currentName string

	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		name := extractName(key)
		if name == "" {
			continue
		}

		if name != currentName {
			if currentService != nil {
				services = append(services, currentService)
			}
			currentName = name
			currentService = &Service{Name: name}
		}

		parseField(currentService, key, string(iter.Value()))
	}
	if currentService != nil {
		services = append(services, currentService)
	}

	return services, nil
}

func (sm *ServiceManager) DeleteService(name string) error {
	prefix := makePrefix(name)
	upperBound := makeUpperBound(name)

	if err := sm.db.DeleteRange(prefix, upperBound, pebble.Sync); err != nil {
		return err
	}

	// Also clean up the service directory (logs, pids)
	serviceDir := sm.GetServiceDir(name)
	return os.RemoveAll(serviceDir)
}

// Helper functions

func makeKey(name, field string) []byte {
	return []byte(fmt.Sprintf("%s%s%s%s%s", prefixService, separator, name, separator, field))
}

func makePrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s%s%s%s", prefixService, separator, name, separator))
}

func makeUpperBound(name string) []byte {
	return []byte(fmt.Sprintf("%s%s%s;", prefixService, separator, name))
}

func extractName(key string) string {
	parts := strings.Split(key, separator)
	if len(parts) < 3 {
		return ""
	}
	return parts[1]
}

func parseField(s *Service, key, val string) {
	parts := strings.Split(key, separator)
	if len(parts) < 3 {
		return
	}
	field := parts[2]

	switch field {
	case fieldCommand:
		s.Command = val
	case fieldStatus:
		s.Status = val
	case fieldPID:
		s.PID, _ = strconv.Atoi(val)
	case fieldCreatedAt:
		s.CreatedAt, _ = time.Parse(time.RFC3339, val)
	case fieldLastStarted:
		s.LastStarted, _ = time.Parse(time.RFC3339, val)
	case fieldLogFile:
		s.LogFile = val
	}
}

package daemon

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StartLogRotationRoutine starts a background routine to handle log rotation.
func (d *Daemon) StartLogRotationRoutine() {
	// Check every hour
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		// Perform an initial check on startup
		d.performLogRotation()
		for range ticker.C {
			d.performLogRotation()
		}
	}()
}

func (d *Daemon) performLogRotation() {
	services, err := d.serviceManager.ListServices()
	if err != nil {
		log.Printf("LogRotation: Failed to list services: %v", err)
		return
	}

	for _, s := range services {
		// Skip if log file doesn't exist
		info, err := os.Stat(s.LogFile)
		if os.IsNotExist(err) {
			continue
		}

		// Check if rotation is needed (if log file was modified before today)
		lastModDate := info.ModTime().Format("2006-01-02")
		today := time.Now().Format("2006-01-02")

		if lastModDate != today {
			archiveFile := s.LogFile + "." + lastModDate

			// Avoid overwriting existing archive if rotation runs multiple times
			if _, err := os.Stat(archiveFile); err == nil {
				continue
			}

			// 1. Copy
			if err := copyFile(s.LogFile, archiveFile); err != nil {
				log.Printf("LogRotation: Failed to copy log for %s: %v", s.Name, err)
				continue
			}

			// 2. Truncate (clear original file)
			if err := os.Truncate(s.LogFile, 0); err != nil {
				log.Printf("LogRotation: Failed to truncate log for %s: %v", s.Name, err)
				continue
			}

			log.Printf("LogRotation: Rotated log for %s to %s", s.Name, archiveFile)
		}

		// 3. Clean old logs (keep 7 days)
		d.cleanOldLogs(filepath.Dir(s.LogFile), 14)
	}
}

func (d *Daemon) cleanOldLogs(dir string, daysRetain int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -daysRetain)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Expected format: service.log.2023-11-29
		if !strings.Contains(name, "service.log.") {
			continue
		}

		// Extract date part
		parts := strings.Split(name, "service.log.")
		if len(parts) < 2 {
			continue
		}
		dateStr := parts[1]

		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				log.Printf("LogRotation: Failed to remove old log %s: %v", name, err)
			} else {
				log.Printf("LogRotation: Removed old log %s", name)
			}
		}
	}
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}
	return nil
}

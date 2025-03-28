package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tangthinker/controlman/internal/client"
	"github.com/tangthinker/controlman/internal/daemon"
)

func main() {
	daemonMode := flag.Bool("daemon", false, "Run in daemon mode")
	flag.Parse()

	if *daemonMode {
		runDaemon()
	} else {
		runClient()
	}
}

func runDaemon() {
	d, err := daemon.NewDaemon()
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down daemon...")
		os.Exit(0)
	}()

	if err := d.Run(); err != nil {
		log.Fatalf("Daemon error: %v", err)
	}
}

func runClient() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	c, err := client.NewClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	command := os.Args[1]
	switch command {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("Usage: controlman add <name> <command>")
			return
		}
		err = c.AddService(os.Args[2], os.Args[3])
		if err != nil {
			log.Fatalf("Failed to add service: %v", err)
		}
		fmt.Printf("Service '%s' added successfully\n", os.Args[2])

	case "stop":
		if len(os.Args) < 3 {
			fmt.Println("Usage: controlman stop <name>")
			return
		}
		err = c.StopService(os.Args[2])
		if err != nil {
			log.Fatalf("Failed to stop service: %v", err)
		}
		fmt.Printf("Service '%s' stopped successfully\n", os.Args[2])

	case "start":
		if len(os.Args) < 3 {
			fmt.Println("Usage: controlman start <name>")
			return
		}
		err = c.StartService(os.Args[2])
		if err != nil {
			log.Fatalf("Failed to start service: %v", err)
		}
		fmt.Printf("Service '%s' started successfully\n", os.Args[2])

	case "logs":
		if len(os.Args) < 3 {
			fmt.Println("Usage: controlman logs <name>")
			return
		}
		logs, err := c.GetLogs(os.Args[2])
		if err != nil {
			log.Fatalf("Failed to get logs: %v", err)
		}
		fmt.Printf("Logs for service '%s':\n%s\n", os.Args[2], logs)
		return

	case "list":
		services, err := c.ListServices()
		if err != nil {
			log.Fatalf("Failed to list services: %v", err)
		}
		if len(services) == 0 {
			fmt.Println("No services found")
			return
		}
		// 打印表头
		fmt.Printf("%-20s %-10s %-8s %-25s %-19s %-50s\n", "NAME", "STATUS", "PID", "CREATED", "LAST START", "COMMAND")
		// 打印服务信息
		for _, s := range services {
			pid := int(s["pid"].(float64))
			fmt.Printf("%-20s %-10s %-8d %-25s %-19s %-50s\n",
				s["name"],
				s["status"],
				pid,
				formatTime(s["created_at"].(string)),
				formatTime(s["last_start"].(string)),
				s["command"])
		}
		return

	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Usage: controlman delete <name>")
			return
		}
		err = c.DeleteService(os.Args[2])
		if err != nil {
			log.Fatalf("Failed to delete service: %v", err)
		}
		fmt.Printf("Service '%s' deleted successfully\n", os.Args[2])

	default:
		printUsage()
		return
	}
}

func formatTime(timeStr string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format("2006-01-02 15:04:05")
}

func printUsage() {
	fmt.Println(`Usage: controlman <command> [arguments]

Commands:
    add <name> <command>    Add a new service
    stop <name>            Stop a service
    start <name>           Start a service
    logs <name>            View service logs
    list                   List all services
    delete <name>          Delete a service
    -daemon               Run in daemon mode`)
}

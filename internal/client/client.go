package client

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

type Client struct {
	socketPath string
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

func NewClient() (*Client, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &Client{
		socketPath: filepath.Join(homeDir, ".controlman", "controlman.sock"),
	}, nil
}

func (c *Client) sendCommand(cmd Command) (*Response, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %v", err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	var response Response
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return &response, nil
}

func (c *Client) AddService(name, command string) error {
	cmd := Command{
		Action:  "add",
		Name:    name,
		Command: command,
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf(resp.Message)
	}

	return nil
}

func (c *Client) StopService(name string) error {
	cmd := Command{
		Action: "stop",
		Name:   name,
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf(resp.Message)
	}

	return nil
}

func (c *Client) StartService(name string) error {
	cmd := Command{
		Action: "start",
		Name:   name,
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf(resp.Message)
	}

	return nil
}

func (c *Client) GetLogs(name string) (string, error) {
	cmd := Command{
		Action: "logs",
		Name:   name,
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf(resp.Message)
	}

	logs, ok := resp.Data.(string)
	if !ok {
		return "", fmt.Errorf("invalid log data type")
	}

	return logs, nil
}

func (c *Client) ListServices() ([]map[string]interface{}, error) {
	cmd := Command{
		Action: "list",
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf(resp.Message)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response data type")
	}

	services := make([]map[string]interface{}, len(data))
	for i, item := range data {
		service, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid service data type")
		}
		services[i] = service
	}

	return services, nil
}

func (c *Client) DeleteService(name string) error {
	cmd := Command{
		Action: "delete",
		Name:   name,
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf(resp.Message)
	}

	return nil
}

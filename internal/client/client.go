package client

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host     string
	Port     uint16
	Username string
	Password string
	Timeout  time.Duration
}

type Client struct {
	config  Config
	client  *ssh.Client
	session *ssh.Session
	stdout  *bufio.Reader
	stdin   io.WriteCloser
}

func (c *Client) Connect() error {
	config := &ssh.ClientConfig{
		User: c.config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.config.Timeout,
	}
	client, err := ssh.Dial("tcp", c.config.Host, config)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("create session: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("create stdout pipe: %w", err)
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("create stdin pipe: %w", err)
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("vt100", 80, 160, modes); err != nil {
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("request PTY: %w", err)
	}
	if err := session.Shell(); err != nil {
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("start shell: %w", err)
	}
	c.client = client
	c.session = session
	c.stdin = stdin
	c.stdout = bufio.NewReader(stdout)
	return nil
}

func (c *Client) Close() {
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.session != nil {
		_ = c.session.Close()
	}
	if c.client != nil {
		_ = c.client.Close()
	}
}

func (c *Client) Execute(command string) (string, error) {
	if c.session == nil {
		return "", fmt.Errorf("execute: client is not connected")
	}
	if _, err := c.stdin.Write([]byte(command + "\n")); err != nil {
		return "", fmt.Errorf("write command: %w", err)
	}
	builder := strings.Builder{}
	for {
		output, err := c.stdout.ReadByte()
		if err != nil {
			return "", fmt.Errorf("read output: %w", err)
		}
		builder.WriteByte(output)
	}
	return "", nil
}

func (c *Client) Read() (string, error) {
	if c.stdout == nil {
		return "", fmt.Errorf("read: client is not connected")
	}
	output, err := c.stdout.ReadString('>')
	if err != nil {
		return "", fmt.Errorf("read output: %w", err)
	}
	return output, nil
}

func (c *Client) Configure(commands ...string) error {
	for i, command := range commands {
		if _, err := c.Execute(command); err != nil {
			return fmt.Errorf("configure command %d: %w", i+1, err)
		}
	}
	return nil
}

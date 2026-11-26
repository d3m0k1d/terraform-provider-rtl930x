package client

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Host     string
	Port     uint16
	Username string
	Password string
	Hostname string
	Timeout  time.Duration
}

type Client struct {
	config  Config
	client  *ssh.Client
	session *ssh.Session
	stdout  *bufio.Reader
	stdin   io.WriteCloser
	prompt  *regexp.Regexp
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
	client, err := ssh.Dial("tcp", net.JoinHostPort(c.config.Host, fmt.Sprintf("%d", c.config.Port)), config)
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
	if _, err := c.Read(""); err != nil {
		c.Close()
		return fmt.Errorf("read initial prompt: %w", err)
	}
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
	return c.Read(command)
}
func (c *Client) Read(command string) (string, error) {
	if c.stdout == nil || c.prompt == nil {
		return "", fmt.Errorf("read: client is not initialized")
	}

	var output strings.Builder
	var line strings.Builder

	for {
		b, err := c.stdout.ReadByte()
		if err != nil {
			return "", fmt.Errorf("read output: %w", err)
		}

		switch b {
		case '\r':
			continue

		case '\n':
			output.WriteString(line.String())
			output.WriteByte('\n')
			line.Reset()

		default:
			line.WriteByte(b)
		}

		if b == '#' || b == '>' {
			if c.prompt.MatchString(strings.TrimSpace(line.String())) {
				break
			}
		}
	}

	result := strings.Trim(output.String(), "\r\n")

	first, rest, _ := strings.Cut(result, "\n")
	if command != "" && strings.TrimSpace(first) == command {
		result = rest
	}

	return result, nil
}

func (c *Client) Configure(commands ...string) error {
	for i, command := range commands {
		if _, err := c.Execute(command); err != nil {
			return fmt.Errorf("configure command %d: %w", i+1, err)
		}
	}
	return nil
}
func New(config Config) *Client {
	promptPattern := `^[[:alnum:]_.-]+(?:\([^\r\n]*\))?[#>]$`
	if config.Hostname != "" {
		promptPattern = `^` + regexp.QuoteMeta(config.Hostname) + `(?:\([^\r\n]*\))?[#>]$`
	}

	return &Client{
		config: config,
		prompt: regexp.MustCompile(promptPattern),
	}
}

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/d3m0k1d/terraform-provider-rtl930x/internal/client"
)

func main() {
	port, err := strconv.ParseUint(os.Getenv("RTL930X_PORT"), 10, 16)
	if err != nil {
		fatal("parse port", err)
	}
	timeout, err := time.ParseDuration(os.Getenv("RTL930X_TIMEOUT"))
	if err != nil {
		fatal("parse timeout", err)
	}

	c := client.New(client.Config{
		Host:     os.Getenv("RTL930X_HOST"),
		Port:     uint16(port),
		Username: os.Getenv("RTL930X_USERNAME"),
		Password: os.Getenv("RTL930X_PASSWORD"),
		Timeout:  timeout,
	})
	defer c.Close()

	if err := c.Connect(); err != nil {
		fatal("connect", err)
	}

	output, err := c.Execute("show vlan")
	if err != nil {
		fatal("execute show vlan", err)
	}
	fmt.Print(output)
}

func fatal(operation string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", operation, err)
	os.Exit(1)
}

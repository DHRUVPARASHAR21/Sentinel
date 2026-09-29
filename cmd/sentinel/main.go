package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/sentinel/sentinel/internal/control"
	"os"
	"strconv"
	"time"
)

const version = "0.1.0-dev"

func main() {
	var socket string
	flag.StringVar(&socket, "socket", "/run/sentinel/sentinel.sock", "Sentinel control socket")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	if args[0] == "version" {
		fmt.Println(version)
		return
	}
	op := args[0]
	req := control.Request{Version: control.Version, Operation: op}
	switch op {
	case "status", "ps", "doctor":
	case "inspect":
		if len(args) != 2 {
			usage()
			os.Exit(2)
		}
		pid, e := strconv.Atoi(args[1])
		if e != nil || pid <= 0 {
			fatal("inspect requires a positive PID")
		}
		req.PID = pid
	case "start", "stop", "restart":
		if len(args) != 2 {
			usage()
			os.Exit(2)
		}
		req.Service = args[1]
	case "services":
	default:
		fatal("unknown command " + op)
	}
	var result any
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := control.Call(ctx, socket, req, &result); err != nil {
		fatal(err.Error())
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
func usage() {
	fmt.Fprintln(os.Stderr, "usage: sentinel [--socket PATH] status|ps|inspect PID|services|start NAME|stop NAME|restart NAME|doctor|version")
}
func fatal(s string) { fmt.Fprintln(os.Stderr, "sentinel:", s); os.Exit(1) }

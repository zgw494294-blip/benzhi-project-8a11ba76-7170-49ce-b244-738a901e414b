package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const defaultAddress = "127.0.0.1:19081"

type trackedString struct {
	value string
	set   bool
}

func (v *trackedString) String() string         { return v.value }
func (v *trackedString) Set(value string) error { v.value, v.set = value, true; return nil }

type config struct {
	Address   string
	DataDir   string
	Selfcheck bool
}

func parseConfig(args []string, lookupEnv func(string) string) (config, error) {
	flags := flag.NewFlagSet("oral-history-release", flag.ContinueOnError)
	address := trackedString{value: defaultAddress}
	flags.Var(&address, "addr", "监听地址，仅允许回环地址")
	dataDir := flags.String("data", "./data", "本地持久化目录")
	selfcheck := flags.Bool("selfcheck", false, "运行有界端到端自检后退出")
	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		return config{}, fmt.Errorf("不接受位置参数")
	}
	resolved := address.value
	if !address.set {
		if portText := strings.TrimSpace(lookupEnv("PORT")); portText != "" {
			port, err := strconv.Atoi(portText)
			if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != portText {
				return config{}, fmt.Errorf("PORT 必须是 1 到 65535 的纯数字端口")
			}
			resolved = net.JoinHostPort("127.0.0.1", portText)
		}
	}
	if err := validateAddress(resolved); err != nil {
		return config{}, err
	}
	if strings.TrimSpace(*dataDir) == "" {
		return config{}, fmt.Errorf("data 目录不能为空")
	}
	return config{Address: resolved, DataDir: *dataDir, Selfcheck: *selfcheck}, nil
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("addr 必须是 host:port：%w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("addr 端口必须在 1 到 65535 之间")
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("addr 只允许 127.0.0.1、localhost 或其他回环 IP")
	}
	return nil
}

func loadConfig() (config, error) { return parseConfig(os.Args[1:], os.Getenv) }

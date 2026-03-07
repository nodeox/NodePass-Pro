package main

import (
	"flag"
	"fmt"
	"log"

	"nodepass-pro/nodeclient/internal/agent"
	"nodepass-pro/nodeclient/internal/config"
)

func main() {
	var (
		configPath  = flag.String("config", "configs/config.yaml", "配置文件路径")
		hubURL      = flag.String("hub-url", "", "覆盖配置中的 hub_url")
		token       = flag.String("token", "", "覆盖配置中的 node_token")
		showVersion = flag.Bool("version", false, "显示客户端版本")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println(agent.Version())
		return
	}

	cfg, err := config.Load(*configPath, config.CLIOverrides{
		HubURL: *hubURL,
		Token:  *token,
	})
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	clientAgent := agent.NewAgent(cfg)

	if err := clientAgent.Start(); err != nil {
		log.Fatalf("Agent 运行失败: %v", err)
	}
}

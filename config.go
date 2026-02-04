package main

import (
	"log"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Config represents options given in the environment
type Config struct {
	SessionExpiration int //in minutes; default: 60

	SQLDriver string //required
	SQLDSN    string //required

	ListenAddr string //addr format used for net.Dial; required
	Prefix     string //url prefix to mount api to without trailing slash

	AIEndpoint                string //AI backend URL; required
	AIModel                   string //AI model name; required
	SummaryAIEndpoint         string //Summary AI backend URL; required
	SummaryAIModel            string //Summary AI model name; required
	ConversationCacheMaxBytes int    //Max size of conversation LRU cache in bytes; default: 10485760 (10MB)
}

var config = &Config{}

func checkEmpty(val, name string) {
	if val == "" {
		log.Fatalf("INVENTORY_%s must be configured\n", name)
	}
}

func init() {
	err := envconfig.Process("INVENTORY", config)
	if err != nil {
		log.Fatalln("Error reading configuration from environment:", err)
	}

	if config.SessionExpiration == 0 {
		config.SessionExpiration = 60
	}

	if config.AIEndpoint == "" {
		log.Fatalln("INVENTORY_AIENDPOINT must be configured")
	}

	if config.AIModel == "" {
		log.Fatalln("INVENTORY_AIMODEL must be configured")
	}

	if config.SummaryAIEndpoint == "" {
		log.Fatalln("INVENTORY_SUMMARYAIENDPOINT must be configured")
	}

	if config.SummaryAIModel == "" {
		log.Fatalln("INVENTORY_SUMMARYAIMODEL must be configured")
	}

	if config.ConversationCacheMaxBytes == 0 {
		config.ConversationCacheMaxBytes = 10485760 // 10MB
	}

	checkEmpty(config.SQLDriver, "SQLDRIVER")
	checkEmpty(config.SQLDSN, "SQLDSN")

	if config.SQLDriver == "mysql" && !strings.Contains(config.SQLDSN, "?parseTime=true") {
		log.Fatalln("mysql DSN must contain \"?parseTime=true\"")
	}

	checkEmpty(config.ListenAddr, "LISTENADDR")
}

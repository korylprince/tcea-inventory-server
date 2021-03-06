package main

import (
	"log"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

//Config represents options given in the environment
type Config struct {
	SessionExpiration int //in minutes; default: 60

	SQLDriver string //required
	SQLDSN    string //required

	ListenAddr string //addr format used for net.Dial; required
	Prefix     string //url prefix to mount api to without trailing slash
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

	checkEmpty(config.SQLDriver, "SQLDRIVER")
	checkEmpty(config.SQLDSN, "SQLDSN")

	if config.SQLDriver == "mysql" && !strings.Contains(config.SQLDSN, "?parseTime=true") {
		log.Fatalln("mysql DSN must contain \"?parseTime=true\"")
	}

	checkEmpty(config.ListenAddr, "LISTENADDR")
}

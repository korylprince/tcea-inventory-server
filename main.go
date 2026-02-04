package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/korylprince/tcea-inventory-server/httpapi"
)

func main() {
	db, err := sql.Open(config.SQLDriver, config.SQLDSN)
	if err != nil {
		log.Fatalln("Could not open database:", err)
	}

	s := httpapi.NewMemorySessionStore(time.Minute * time.Duration(config.SessionExpiration))

	chatCfg := &httpapi.ChatConfig{
		AIEndpoint:        config.AIEndpoint,
		AIModel:           config.AIModel,
		SummaryAIEndpoint: config.SummaryAIEndpoint,
		SummaryAIModel:    config.SummaryAIModel,
		CacheMaxBytes:     config.ConversationCacheMaxBytes,
	}

	r := httpapi.NewRouter(os.Stdout, s, db, chatCfg)

	chain := handlers.CompressHandler(handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Accept", "Content-Type", "Origin", "X-Session-Key"}),
	)(http.StripPrefix(config.Prefix, r)))

	log.Println("Listening on:", config.ListenAddr)
	log.Println(http.ListenAndServe(config.ListenAddr, chain))
}

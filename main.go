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

	r := httpapi.NewRouter(os.Stdout, s, db)

	chain := handlers.CompressHandler(http.StripPrefix(config.Prefix, r))

	log.Println("Listening on:", config.ListenAddr)
	log.Println(http.ListenAndServe(config.ListenAddr, chain))
}

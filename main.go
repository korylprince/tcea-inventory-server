package main

import (
	"context"
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

	s := httpapi.NewMemorySessionStore(time.Hour * time.Duration(config.SessionDuration))

	ctx := context.Background()

	r := httpapi.NewRouter(ctx, s, db)

	chain := handlers.CombinedLoggingHandler(os.Stdout,
		handlers.CompressHandler(
			http.StripPrefix(config.Prefix, r)))

	log.Println("Listening on:", config.ListenAddr)
	log.Println(http.ListenAndServe(config.ListenAddr, chain))
}

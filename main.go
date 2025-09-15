package main

import (
	"flag"
	"log"
	"database/sql"

	_ "modernc.org/sqlite"
	"github.com/SvenDH/go-card-engine/server"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()

	db, err := sql.Open("sqlite", "./db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	
	repo := server.NewRepository(db)
	broker := server.NewMemoryBroker()
	wsServer := server.NewWebsocketServer(broker, repo)
	router := server.NewRouter(*addr, broker, repo, wsServer)
	
	router.Run()
}
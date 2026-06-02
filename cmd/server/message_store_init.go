package main

import (
	"log"

	sqlitestore "github.com/camronwood/neural-junkie/internal/store/sqlite"
)

func initMessageStore() {
	store, err := sqlitestore.Open("")
	if err != nil {
		log.Printf("SQLite message store unavailable: %v", err)
		return
	}
	if chatHub != nil {
		chatHub.SetPersistentMessageStore(store)
		log.Println("SQLite message store enabled (~/.neural-junkie/messages.db)")
	}
}

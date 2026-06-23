package main

import (
	"log"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
	sqlitestore "github.com/camronwood/neural-junkie/internal/store/sqlite"
)

type sqliteMessageStore struct {
	*sqlitestore.Store
}

func (a *sqliteMessageStore) SearchMessages(opts hub.MessageSearchOptions) ([]*protocol.Message, error) {
	return a.Store.SearchWithOptions(sqlitestore.SearchOptions{
		Channel: opts.Channel,
		Query:   opts.Query,
		Limit:   opts.Limit,
		Before:  opts.Before,
	})
}

func initMessageStore() {
	store, err := sqlitestore.Open("")
	if err != nil {
		log.Printf("SQLite message store unavailable: %v", err)
		return
	}
	if chatHub != nil {
		chatHub.SetPersistentMessageStore(&sqliteMessageStore{Store: store})
		log.Println("SQLite message store enabled (~/.neural-junkie/messages.db)")
	}
}

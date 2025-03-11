package main

import (
	"log"

	"github.com/keshavrathinvael/Big-O-Solution/internal"
	"github.com/keshavrathinvael/Big-O-Solution/internal/storage"
)

func main() {
	poolManager := storage.NewPoolManager()
	segHashTable := storage.NewSegmentedHashTable(10, 256)
	server := internal.CreateServer(segHashTable, poolManager)
	server.SetReady(true)
	err := server.Start(5555)
	if err != nil {
		log.Fatal(err)
	}
}

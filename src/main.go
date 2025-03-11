package main

import (
	"log"

	"github.com/keshavrathinvael/Big-O-Solution/internal"
	"github.com/keshavrathinvael/Big-O-Solution/internal/storage"
)

func main() {
	println("Starting Pandora's Data Hub...")

	storeSize := uint64(3 * 1024 * 1024 * 1024)
	poolManager := storage.NewPoolManager()
	segHashTable := storage.NewSegmentedHashTable(16, storeSize)
	server := internal.CreateServer(segHashTable, poolManager)
	server.SetReady(true)
	err := server.Start(5555)
	if err != nil {
		log.Fatal(err)
	}
}

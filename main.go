package main

import (
	"log"
	"net/http"

	user "svc-opt/usecase"

	_ "net/http/pprof"
	"svc-opt/lib/batcher"
	"svc-opt/lib/ioc"
	"time" // Import pprof for profiling
)

func initPprof() {
	go func() {
		log.Println("Starting pprof on :6060")
		log.Println(http.ListenAndServe("localhost:6060", nil)) // pprof server
	}()
}

func initBatcher() {
	ioc.BatchIOC = batcher.NewBatcher(
		batcher.WithCleanupInterval(5*time.Second),
		batcher.WithStaleDuration(5*time.Second),
	)

	batchListener := batcher.NewListener(
		ioc.GET_USER_BY_IDS,
		batcher.WithMaxRequests(ioc.BATCH_MAX_REQ),
		batcher.WithMaxWait(ioc.BATCH_MAX_WAIT),
		batcher.WithRunner(user.BatchRunner),
	)
	ioc.BatchIOC.AddListener(batchListener)
}

func main() {
	// Init pprof
	initPprof()

	// Init batcher
	// initBatcher()

	// Init http
	http.HandleFunc("/user/", user.GetUserByID)
	http.HandleFunc("/user-batch/", user.GetUserByIDBatch)
	log.Println("Starting main service on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

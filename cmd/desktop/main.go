package main

import (
	"context"
	"flag"
	"log"
	"time"
	"vocabulary/internal/storage"
	"vocabulary/internal/ui"
)

func main() {
	storage, err := storage.Open(context.Background(), STORAGE_FILE_PATH)

	if err != nil {
		log.Fatal(err)
	}

	defer storage.Close()

	appImpl := &loadAllFile{
		storage: storage,
	}

	defer appImpl.exit()

	flag.Parse()

	if flag.NArg() == 1 {
		appImpl.OpenFile(flag.Arg(0))
	} else {
		appImpl.OpenLast()
	}

	ui.Run(appImpl)

	storage.EraseOutdatedData(
		context.Background(),
		MAX_LESSONS_COUNT_TO_STORE_PROGRESS,
		time.Now().Add(TIME_TO_STORE_LESSONS_PROGRESS*-1),
	)
}

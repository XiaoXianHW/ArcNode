package main

import (
	"flag"
	"log"

	"github.com/XiaoXianHW/ArcNode/gateway/api"
	"github.com/XiaoXianHW/ArcNode/gateway/category"
	"github.com/XiaoXianHW/ArcNode/gateway/config"
	"github.com/XiaoXianHW/ArcNode/gateway/storage"
	"github.com/XiaoXianHW/ArcNode/gateway/web"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	store, err := storage.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.Close()

	classifier := category.New(cfg.Rules())
	if custom, err := store.CustomKeywordMap(); err == nil {
		classifier.SetCustom(custom)
	}

	if res, err := store.ReclassifyAll(classifier); err != nil {
		log.Printf("reclassify on startup failed: %v", err)
	} else if res.SegmentsUpdated > 0 || res.EventsUpdated > 0 {
		log.Printf("reclassify: segments %d/%d, events %d/%d updated",
			res.SegmentsUpdated, res.SegmentsScanned, res.EventsUpdated, res.EventsScanned)
	}

	server := &api.Server{
		Store:      store,
		Classifier: classifier,
		Token:      cfg.Token,
		SegmentGap: cfg.SegmentGapSeconds,
		WebFS:      web.FS(),
	}

	log.Printf("ArcNode gateway listening on %s", cfg.Listen)
	if err := server.Router().Run(cfg.Listen); err != nil {
		log.Fatalf("server: %v", err)
	}
}

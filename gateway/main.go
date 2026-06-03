package main

import (
	"flag"
	"log"
	"time"

	"github.com/XiaoXianHW/ArcNode/gateway/api"
	"github.com/XiaoXianHW/ArcNode/gateway/category"
	"github.com/XiaoXianHW/ArcNode/gateway/config"
	"github.com/XiaoXianHW/ArcNode/gateway/storage"
	"github.com/XiaoXianHW/ArcNode/gateway/web"
)

// startRetention launches a background loop that, when a positive retention
// window is configured, prunes events/segments older than the cutoff on
// startup and then once a day, reclaiming space afterwards.
func startRetention(store *storage.Store, retentionDays int64) {
	if retentionDays <= 0 {
		return
	}
	prune := func() {
		cutoff := time.Now().AddDate(0, 0, int(-retentionDays)).Unix()
		ev, seg, err := store.PruneOlderThan(cutoff)
		if err != nil {
			log.Printf("retention prune failed: %v", err)
			return
		}
		if ev > 0 || seg > 0 {
			log.Printf("retention prune: removed %d events, %d segments older than %d days", ev, seg, retentionDays)
			if err := store.Vacuum(); err != nil {
				log.Printf("retention vacuum failed: %v", err)
			}
		}
	}
	go func() {
		prune()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			prune()
		}
	}()
}

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

	startRetention(store, cfg.RetentionDays)

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

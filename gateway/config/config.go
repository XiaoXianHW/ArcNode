package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Listen            string              `toml:"listen"`
	Token             string              `toml:"token"`
	DBPath            string              `toml:"db_path"`
	SegmentGapSeconds int64               `toml:"segment_gap_seconds"`
	Categories        map[string][]string `toml:"categories"`
}

func defaults() *Config {
	return &Config{
		Listen:            ":8080",
		Token:             "change-me",
		DBPath:            "./gateway.db",
		SegmentGapSeconds: 60,
		Categories:        defaultCategories(),
	}
}

func Load(path string) (*Config, error) {
	cfg := defaults()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if cfg.SegmentGapSeconds <= 0 {
		cfg.SegmentGapSeconds = 60
	}
	if len(cfg.Categories) == 0 {
		cfg.Categories = defaultCategories()
	}
	return cfg, nil
}

func defaultCategories() map[string][]string {
	return map[string][]string{
		"coding": {
			"code", "vscode", "cursor", "zed", "sublime", "atom", "xcode",
			"intellij", "idea", "pycharm", "goland", "webstorm", "clion", "rider", "phpstorm",
			"vim", "nvim", "emacs", "helix",
			"terminal", "iterm", "wezterm", "alacritty", "kitty", "warp",
			"github", "gitlab",
		},
		"gaming": {
			"steam", "epic games", "battle.net", "blizzard",
			"minecraft", "valorant", "counter-strike", "cs2", "dota",
			"overwatch", "genshin", "launcher",
		},
		"video": {
			"bilibili", "youtube", "netflix", "twitch",
			"mpv", "vlc", "potplayer", "iina", "plex",
		},
		"music": {
			"spotify", "apple music", "netease music", "qq music", "cloudmusic",
		},
		"communication": {
			"wechat", "weixin", "qq", "discord", "slack", "telegram", "teams",
			"zoom", "lark", "feishu", "dingtalk",
		},
		"browsing": {
			"chrome", "firefox", "edge", "safari", "brave", "opera", "arc",
		},
		"productivity": {
			"notion", "obsidian", "logseq", "onenote",
			"word", "excel", "powerpoint", "figma", "adobe", "photoshop",
		},
	}
}

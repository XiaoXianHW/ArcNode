package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/XiaoXianHW/ArcNode/gateway/category"
)

// Config is the gateway runtime config. Both legacy and v2 category schemas
// are accepted: the legacy `[categories]` table (flat string arrays) is read
// as process keywords for backward compatibility, while v2 supports
// `[categories.<name>.process]` and `[categories.<name>.title]`.
type Config struct {
	Listen            string                     `toml:"listen"`
	Token             string                     `toml:"token"`
	DBPath            string                     `toml:"db_path"`
	SegmentGapSeconds int64                      `toml:"segment_gap_seconds"`
	Categories        map[string][]string        `toml:"categories"`
	CategoryRules     map[string]CategoryRuleCfg `toml:"category_rules"`
}

// CategoryRuleCfg is the v2 user-facing split schema.
type CategoryRuleCfg struct {
	Process []string `toml:"process"`
	Title   []string `toml:"title"`
}

func defaults() *Config {
	return &Config{
		Listen:            ":8080",
		Token:             "change-me",
		DBPath:            "./gateway.db",
		SegmentGapSeconds: 60,
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
	return cfg, nil
}

// Rules returns the merged rule set: built-in defaults overlaid by the
// user's `[category_rules]` v2 schema overlaid by the legacy `[categories]`
// flat schema (treated as additional process keywords).
func (c *Config) Rules() map[string]category.Rules {
	base := DefaultRules()
	for cat, r := range c.CategoryRules {
		cur := base[cat]
		cur.Process = append(cur.Process, r.Process...)
		cur.Title = append(cur.Title, r.Title...)
		base[cat] = cur
	}
	for cat, list := range c.Categories {
		cur := base[cat]
		cur.Process = append(cur.Process, list...)
		base[cat] = cur
	}
	return base
}

// DefaultRules returns the built-in classifier rules split into
// process_name and window_title keyword sets per category.
//
// Most categories use process-only matching: matching on title is too
// noisy (e.g. a Chrome tab whose title contains "Minecraft" should not
// turn into the gaming category). Categories where title-based
// classification is genuinely useful (e.g. browser tabs showing
// youtube.com counting as video) still keep their title set; the
// classifier upgrades the result from "browsing" → that category when
// the title matches a more specific one.
func DefaultRules() map[string]category.Rules {
	return map[string]category.Rules{
		"terminal": {
			Process: []string{
				"windowsterminal.exe", "windows terminal", "wt.exe",
				"powershell.exe", "powershell", "pwsh.exe", "pwsh",
				"cmd.exe", "conhost.exe", "command prompt",
				"terminal", "terminal.app", "iterm", "iterm2",
				"wezterm", "wezterm-gui", "alacritty", "kitty",
				"warp", "warpterminal", "hyper",
				"tabby", "tabby.exe", "konsole", "gnome-terminal",
				"xfce4-terminal", "tilix", "guake", "xterm",
				"rxvt", "urxvt", "termite", "ghostty", "rio", "contour",
				"deepin-terminal", "qterminal",
			},
		},
		"coding": {
			Process: []string{
				"code.exe", "code - insiders", "code-insiders", "code",
				"vscode", "visual studio code", "codium", "vscodium",
				"cursor.exe", "cursor", "windsurf", "trae.exe", "trae",
				"void.exe", "void", "zed.exe", "zed", "fleet", "jetbrains fleet",
				"idea64.exe", "idea.exe", "idea", "intellij", "intellij idea",
				"pycharm64.exe", "pycharm.exe", "pycharm",
				"goland64.exe", "goland.exe", "goland",
				"webstorm64.exe", "webstorm.exe", "webstorm",
				"clion64.exe", "clion.exe", "clion",
				"phpstorm64.exe", "phpstorm.exe", "phpstorm",
				"rider64.exe", "rider.exe", "rider",
				"rubymine64.exe", "rubymine.exe", "rubymine",
				"datagrip64.exe", "datagrip.exe", "datagrip",
				"appcode", "studio64.exe", "studio.exe", "android studio",
				"mps", "rustrover", "aqua", "writerside",
				"devenv.exe", "visual studio", "vs 2022", "vs 2019",
				"xcode", "xcode-beta", "swift playgrounds",
				"sublime_text.exe", "sublime_text", "sublime text",
				"atom.exe", "atom", "notepad++.exe", "notepad++", "bbedit",
				"textmate", "textedit", "gedit", "geany", "kate", "kwrite", "smultron",
				"brackets", "komodo", "ultraedit",
				"gvim.exe", "gvim", "macvim", "neovim", "nvim.exe", "nvim", "nvim-qt",
				"emacs", "spacemacs", "helix", "hx",
				"kakoune", "micro", "nano",
				"lapce", "lite-xl", "kiro", "kiro.exe",
				"eclipse", "netbeans", "spring-tool-suite", "bluej", "drjava", "jgrasp",
				"qt creator", "qtcreator", "code::blocks", "codeblocks", "dev-c++", "devcpp",
				"jupyter", "jupyter-notebook", "jupyter-lab", "jupyterlab", "rstudio",
				"spyder", "anaconda navigator", "matlab", "octave", "stata", "sas",
				"orange3", "knime", "dataiku",
				"unity.exe", "unity", "unityhub", "unrealeditor", "ue4editor", "ue5editor",
				"godot.exe", "godot",
				"rpg maker", "gamemaker", "construct", "defold",
				"flutter", "xamarin",
				"dbeaver.exe", "dbeaver", "navicat", "tableplus", "sequel ace", "sequel pro",
				"heidisql", "mysql workbench", "postman", "insomnia", "bruno", "yaak",
				"hoppscotch", "pgadmin", "redisinsight", "mongodb compass", "studio 3t",
				"github desktop", "gitkraken", "sourcetree", "tower", "fork", "tig",
				"lazygit", "magit", "smartgit", "gitg", "gitup", "sublime merge",
				"docker desktop", "rancher desktop", "podman desktop", "lens",
				"k9s", "minikube",
			},
		},
		"ai_tools": {
			Process: []string{
				"lm studio", "lmstudio", "ollama", "comfyui", "comfyui-portable",
				"automatic1111", "invokeai", "fooocus", "cherry studio", "chatbox",
				"raycast-ai", "msty",
			},
			Title: []string{
				"chatgpt", "chat.openai.com", "claude.ai", "anthropic", "gemini.google.com",
				"perplexity.ai", "huggingface.co", "deepseek.com", "kimi.moonshot.cn",
				"doubao.com", "tongyi.aliyun", "yiyan.baidu", "wenxin.baidu",
				"midjourney.com", "runwayml.com", "pika.art",
				"copilot.microsoft", "github copilot",
			},
		},
		"design": {
			Process: []string{
				"figma.exe", "figma", "figma agent", "sketch", "adobe xd", "framer",
				"principle", "invision", "miro",
				"photoshop.exe", "photoshop", "illustrator.exe", "illustrator",
				"indesign.exe", "indesign", "lightroom.exe", "lightroom",
				"affinity photo", "affinity designer", "affinity publisher",
				"gimp", "krita", "inkscape", "procreate", "clip studio paint", "csp",
				"paint tool sai", "sai2", "medibang", "rebelle", "fresco", "concepts",
				"blender.exe", "blender", "maya", "3ds max", "cinema 4d", "c4d",
				"houdini", "zbrush",
				"substance painter", "substance designer", "marvelous designer", "spline",
				"rhino", "autocad", "fusion 360", "solidworks", "sketchup",
				"davinci resolve", "premiere", "premiere pro", "after effects",
				"final cut", "final cut pro", "motion", "fcpx", "vegas pro",
				"capcut", "剪映", "shotcut", "obs", "obs64.exe", "obs studio",
				"streamlabs", "xsplit",
			},
		},
		"gaming": {
			Process: defaultGamingProcessKeywords(),
		},
		"video": {
			Process: []string{
				"mpv", "vlc", "vlc.exe", "potplayer", "potplayermini64.exe",
				"iina", "plex", "infuse", "kodi", "jellyfin",
				"emby", "stremio", "media player classic", "mpc-hc", "mpc-be",
				"netflix.exe", "movies & tv", "movies and tv",
			},
			Title: []string{
				"youtube.com", " - youtube", "youtube music",
				"bilibili.com", " - bilibili", "哔哩哔哩",
				"twitch.tv", "netflix.com", "primevideo.com", "disneyplus.com",
				"iqiyi.com", "爱奇艺", "youku.com", "优酷",
				"v.qq.com", "腾讯视频", "mgtv.com", "芒果tv", "tv.cctv.com",
				"niconico.jp", "abema.tv",
			},
		},
		"music": {
			Process: []string{
				"spotify.exe", "spotify", "apple music", "itunes", "tidal", "deezer",
				"soundcloud", "qobuz",
				"cloudmusic.exe", "cloudmusic", "netease cloud music", "网易云音乐",
				"qqmusic.exe", "qq music", "qq音乐", "kugou", "kuwo", "酷狗", "酷我", "migu music",
				"foobar2000", "foobar", "musicbee", "winamp", "aimp",
				"audirvana", "roon", "plexamp", "navidrome",
			},
			Title: []string{
				"music.youtube.com", "music.apple.com", "open.spotify.com",
			},
		},
		"communication": {
			Process: []string{
				"wechat.exe", "wechat", "weixin.exe", "weixin", "微信", "qqnt",
				"qq.exe", "tencent qq",
				"discord.exe", "discord", "slack.exe", "slack",
				"telegram.exe", "telegram", "telegramdesktop", "telegramdesktop.exe",
				"teams.exe", "teams", "microsoft teams", "ms-teams.exe",
				"zoom.exe", "zoom", "skype", "signal", "whatsapp", "messenger",
				"facebook messenger",
				"line", "kakaotalk", "viber", "wire", "element", "rocket.chat",
				"lark.exe", "lark", "feishu.exe", "feishu", "飞书",
				"dingtalk.exe", "dingtalk", "钉钉",
				"wxwork.exe", "wecom", "wechat work", "企业微信",
				"thunderbird.exe", "thunderbird", "outlook.exe", "outlook", "mail.exe",
				"spark", "airmail", "newton", "mailspring", "mimestream", "spike", "front",
			},
		},
		"social": {
			Process: []string{
				"weibo.exe", "douyin.exe", "tiktok.exe", "rednote", "xiaohongshu",
			},
			Title: []string{
				"twitter.com", " / x", "x.com", "instagram.com", "facebook.com",
				"reddit.com", "snapchat", "tiktok.com",
				"douyin.com", "抖音", "xiaohongshu.com", "小红书",
				"weibo.com", "微博", "zhihu.com", "知乎",
				"tieba.baidu.com", "贴吧", "bsky.app", "bluesky",
				"mastodon", "threads.net", "tumblr.com",
				"pinterest.com", "linkedin.com", "v2ex.com", "quora.com",
			},
		},
		"reading": {
			Process: []string{
				"acrord32.exe", "acrobat.exe", "adobe acrobat", "acrobat reader",
				"preview.app", "preview", "skim", "pdf expert",
				"foxitreader.exe", "foxit reader", "foxit pdf",
				"sumatrapdf.exe", "sumatra pdf", "okular", "evince", "zathura",
				"calibre.exe", "calibre", "kindle.exe", "kindle", "amazon kindle",
				"books.app", "apple books", "ibooks",
				"weread.exe", "weread", "微信读书", "duokan", "moon+ reader", "kobo",
				"readest", "koreader", "polar bookshelf",
				"zotero.exe", "zotero", "mendeley", "endnote", "papers",
			},
		},
		"productivity": {
			Process: []string{
				"notion.exe", "notion", "obsidian.exe", "obsidian",
				"logseq.exe", "logseq", "anytype", "roamresearch", "remnote",
				"onenote.exe", "onenote", "evernote.exe", "evernote", "joplin",
				"bear.app", "bear", "craft", "ulysses", "scrivener",
				"typora.exe", "typora", "marktext", "zettlr", "ghostwriter", "ia writer",
				"winword.exe", "microsoft word", "excel.exe", "powerpnt.exe",
				"pages.app", "numbers.app", "keynote.app",
				"wps.exe", "wps office", "金山办公",
				"libreoffice", "openoffice",
				"airtable", "coda", "clickup", "asana", "trello", "monday", "linear",
				"jira", "confluence", "todoist", "things", "omnifocus",
				"reminders", "calendar.app", "fantastical", "cron",
				"notion calendar",
				"raycast", "alfred", "launchbar", "spotlight",
				"explorer.exe", "windows explorer", "file explorer",
				"finder", "finder.app",
				"nautilus", "nemo", "thunar", "pcmanfm",
				"caja", "rox-filer", "krusader", "doublecmd", "totalcmd.exe", "total commander",
				"directory opus", "dopus", "xyplorer", "xyplorerfree",
				"q-dir", "qdir", "freecommander", "multicommander",
			},
		},
		"browsing": {
			Process: []string{
				"chrome.exe", "google chrome", "chrome",
				"firefox.exe", "firefox", "firefox developer edition",
				"msedge.exe", "microsoft edge", "edge",
				"safari", "brave.exe", "brave", "brave browser",
				"opera.exe", "opera", "opera gx", "arc.exe", "arc",
				"vivaldi.exe", "vivaldi", "tor browser", "torbrowser", "thorium",
				"chromium", "ungoogled-chromium", "zen browser",
				"orion", "min", "sidekick",
				"360se.exe", "360chrome.exe", "360 browser", "360极速浏览器",
				"qqbrowser.exe", "qq browser", "qq浏览器",
				"ucbrowser.exe", "uc browser",
				"sogouexplorer.exe", "sogou explorer", "搜狗浏览器",
			},
		},
	}
}

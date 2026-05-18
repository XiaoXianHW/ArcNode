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
			"code.exe", "code - insiders", "code-insiders", "vscode", "visual studio code",
			"codium", "vscodium",
			"cursor", "cursor.exe", "windsurf", "trae", "trae.exe", "void", "void.exe",
			"zed", "zed.exe", "fleet", "jetbrains fleet",
			"idea", "idea64.exe", "intellij", "intellij idea", "pycharm", "pycharm64.exe",
			"goland", "goland64.exe", "webstorm", "webstorm64.exe", "clion", "clion64.exe",
			"phpstorm", "phpstorm64.exe", "rider", "rider64.exe", "rubymine", "rubymine64.exe",
			"datagrip", "datagrip64.exe", "appcode", "studio64.exe", "android studio",
			"mps", "rustrover", "aqua", "writerside",
			"devenv.exe", "visual studio", "vs 2022", "vs 2019",
			"xcode", "xcode-beta", "swift playgrounds",
			"sublime_text", "sublime text", "atom", "notepad++", "bbedit",
			"textmate", "textedit", "gedit", "geany", "kate", "kwrite", "smultron",
			"brackets", "komodo", "ultraedit",
			"vim", "gvim", "macvim", "neovim", "nvim", "nvim-qt", "emacs", "spacemacs",
			"helix", "kakoune", "micro", "nano",
			"lapce", "lite-xl",
			"eclipse", "netbeans", "spring-tool-suite", "bluej", "drjava", "jgrasp",
			"qt creator", "qtcreator", "code::blocks", "codeblocks", "dev-c++", "devcpp",
			"jupyter", "jupyter-notebook", "jupyter-lab", "jupyterlab", "rstudio",
			"spyder", "anaconda navigator", "matlab", "octave", "stata", "sas",
			"orange3", "knime", "dataiku",
			"unity", "unityhub", "unrealeditor", "ue4editor", "ue5editor", "godot",
			"rpg maker", "gamemaker", "construct", "defold",
			"flutter", "xamarin",
			"dbeaver", "navicat", "tableplus", "sequel ace", "sequel pro", "heidisql",
			"mysql workbench", "postman", "insomnia", "bruno", "yaak", "hoppscotch",
			"pgadmin", "redisinsight", "mongodb compass", "studio 3t",
			"terminal", "iterm", "iterm2", "wezterm", "alacritty", "kitty", "warp",
			"windows terminal", "wt.exe", "powershell", "pwsh", "cmd.exe",
			"hyper", "tabby", "konsole", "gnome-terminal", "tilix", "guake", "xterm",
			"rxvt", "urxvt", "termite", "ghostty", "rio", "contour",
			"github desktop", "gitkraken", "sourcetree", "tower", "fork", "tig",
			"lazygit", "magit", "smartgit", "gitg", "gitup", "sublime merge",
			"docker desktop", "rancher desktop", "podman desktop", "lens",
			"k9s", "minikube",
		},
		"gaming": {
			"steam", "steamwebhelper", "epic games", "epicgameslauncher", "battle.net",
			"blizzard", "battle.net.exe", "origin", "ea app", "eaapp", "eadesktop",
			"uplay", "ubisoft connect", "upc.exe", "gog galaxy", "rockstar games launcher",
			"riot client", "riotclientux", "riotclientservices", "leagueclient",
			"bethesda.net launcher", "amazon games",
			"itch.io", "playnite", "lutris", "heroic games launcher",
			"xbox", "xboxapp", "wegame", "tencent games", "wegameapp",
			"league of legends", "leagueclient.exe", "league of legends.exe",
			"dota 2", "dota2.exe", "honor of kings", "smite", "heroes of the storm",
			"valorant", "valorant-win64-shipping", "counter-strike", "cs2", "cs:go",
			"csgo", "csgo.exe", "rainbow six", "r6", "overwatch", "overwatch.exe",
			"apex legends", "r5apex", "call of duty", "modernwarfare", "modernwarfareii",
			"warzone", "destiny 2", "destiny2", "halo", "halo infinite", "deltaforce",
			"escape from tarkov", "eft", "pubg", "tslgame", "fortnite", "fortniteclient",
			"the finals", "battlefield", "bf2042", "splitgate", "paladins", "deadlock",
			"crossfire", "rust", "rustclient", "dayz", "arma", "arma3",
			"insurgency", "hell let loose", "squad", "war thunder", "warthunder",
			"world of tanks", "wot", "world of warships", "wows",
			"world of warcraft", "wow.exe", "final fantasy", "ffxiv", "ffxiv_dx11",
			"lost ark", "guild wars 2", "gw2", "elder scrolls online", "eso",
			"black desert", "blade & soul", "tera", "new world", "throne and liberty",
			"swtor", "ragnarok",
			"genshin impact", "genshinimpact.exe", "yuanshen.exe", "honkai", "honkai impact",
			"honkai: star rail", "honkaistarrail.exe", "zenless zone zero", "zenlesszonezero",
			"wuthering waves", "wuwa", "punishing gray raven", "tower of fantasy",
			"snowbreak", "girls' frontline 2",
			"the witcher", "witcher3", "cyberpunk2077", "cyberpunk", "elden ring", "eldenring",
			"dark souls", "darksouls", "sekiro", "bloodborne", "monster hunter",
			"monsterhunterrise", "monsterhunterworld", "dragon's dogma", "hogwarts legacy",
			"hogwartslegacy", "rdr2", "red dead redemption", "gta5", "gtav", "gta_sa",
			"baldur's gate 3", "bg3", "starfield", "skyrim", "fallout4", "fallout76",
			"persona", "persona5", "persona3", "yakuza", "like a dragon",
			"silent hill", "resident evil", "biohazard", "diablo iv", "diablo4",
			"path of exile", "poe", "marvel rivals",
			"civilization", "civilizationvi", "civ7", "civ6",
			"stellaris", "europa universalis", "hearts of iron", "crusader kings", "ck3",
			"age of empires", "aoe4", "aoe2", "starcraft", "warcraft iii", "warcraft3",
			"company of heroes", "total war", "warhammer", "totalwarhammer",
			"factorio", "rimworld", "dwarf fortress",
			"minecraft", "minecraftlauncher", "terraria", "starbound", "valheim",
			"the forest", "sons of the forest", "subnautica", "raft", "no man's sky",
			"7 days to die", "satisfactory", "core keeper", "vintage story",
			"palworld", "enshrouded", "stranded deep", "green hell",
			"phasmophobia", "lethal company", "content warning", "deep rock galactic",
			"left 4 dead", "back 4 blood", "vermintide", "darktide", "warhammer 40k",
			"helldivers", "helldivers 2", "among us", "fall guys", "human fall flat",
			"gang beasts", "stumble guys", "stick fight",
			"forza horizon", "forza motorsport", "f1 2024", "f12024", "f12023",
			"gran turismo", "asphalt", "need for speed", "burnout",
			"fifa", "ea sports fc", "fc25", "nba 2k", "nba2k", "madden", "pes",
			"hollow knight", "celeste", "hades", "stardew valley", "stardewvalley",
			"cuphead", "ori", "undertale", "deltarune", "katana zero",
			"limbo", "inside", "vampire survivors", "balatro",
			"slay the spire", "into the breach", "ftl", "dead cells",
			"animal crossing", "splatoon",
			"arknights", "uma musume", "blue archive",
			"granblue fantasy relink", "blue protocol",
			"naraka bladepoint", "naraka", "delta force", "perfect world",
			"yuzu", "ryujinx", "citra", "dolphin", "pcsx2", "rpcs3", "ppsspp",
			"retroarch", "duckstation", "shadps4", "mednafen",
			"vrchat", "vrchat.exe", "chillout vr",
		},
		"ai_tools": {
			"chatgpt", "claude", "claude.ai", "gemini", "bard", "copilot", "github copilot",
			"perplexity", "anthropic", "openai", "huggingface", "deepseek", "kimi",
			"doubao", "tongyi", "wenxin", "yiyan", "ollama", "lm studio", "lmstudio",
			"comfyui", "stable diffusion", "automatic1111", "invokeai", "fooocus",
			"midjourney", "runway", "pika", "cherry studio", "chatbox",
		},
		"design": {
			"figma", "sketch", "adobe xd", "framer", "principle", "invision", "miro",
			"photoshop", "illustrator", "indesign", "lightroom", "affinity photo",
			"affinity designer", "affinity publisher", "gimp", "krita", "inkscape",
			"procreate", "clip studio paint", "csp", "paint tool sai", "sai2",
			"medibang", "rebelle", "fresco", "concepts",
			"blender", "maya", "3ds max", "cinema 4d", "c4d", "houdini", "zbrush",
			"substance painter", "substance designer", "marvelous designer", "spline",
			"rhino", "autocad", "fusion 360", "solidworks", "sketchup",
			"davinci resolve", "premiere", "premiere pro", "after effects", "final cut",
			"final cut pro", "motion", "fcpx", "vegas pro", "capcut", "剪映", "shotcut",
			"obs", "obs studio", "streamlabs", "xsplit",
		},
		"video": {
			"bilibili", "哔哩哔哩", "youtube", "youtube music", "netflix", "twitch",
			"prime video", "disney+", "disneyplus", "hbo max", "hulu",
			"iqiyi", "爱奇艺", "youku", "优酷", "tencent video", "腾讯视频",
			"mango tv", "芒果tv", "migu", "viu", "abema", "niconico",
			"mpv", "vlc", "potplayer", "iina", "plex", "infuse", "kodi", "jellyfin",
			"emby", "stremio", "media player classic", "mpc-hc", "mpc-be",
		},
		"music": {
			"spotify", "apple music", "itunes", "tidal", "deezer", "soundcloud",
			"qobuz", "youtube music",
			"netease cloud music", "网易云音乐", "cloudmusic", "qq music", "qq音乐",
			"kugou", "kuwo", "酷狗", "酷我", "migu music",
			"foobar2000", "foobar", "musicbee", "winamp", "aimp",
			"audirvana", "roon", "plexamp", "navidrome",
		},
		"communication": {
			"wechat", "weixin", "微信", "qq", "tencent qq", "qqnt",
			"discord", "slack", "telegram", "telegramdesktop", "teams", "microsoft teams",
			"zoom", "skype", "signal", "whatsapp", "messenger", "facebook messenger",
			"line", "kakaotalk", "viber", "wire", "element", "rocket.chat",
			"lark", "feishu", "飞书", "dingtalk", "钉钉", "wecom", "wechat work", "企业微信",
			"thunderbird", "outlook", "mail", "spark", "airmail", "newton", "mailspring",
			"mimestream", "spike", "front",
		},
		"browsing": {
			"chrome", "google chrome", "firefox", "firefox developer edition", "edge",
			"microsoft edge", "msedge", "safari", "brave", "brave browser", "opera",
			"opera gx", "arc", "vivaldi", "tor browser", "torbrowser", "thorium",
			"chromium", "ungoogled-chromium", "zen browser", "orion", "min", "sidekick",
			"360 browser", "360极速浏览器", "qq browser", "qq浏览器", "uc browser",
			"sogou explorer", "搜狗浏览器",
		},
		"productivity": {
			"notion", "obsidian", "logseq", "anytype", "roamresearch", "remnote",
			"onenote", "evernote", "joplin", "bear", "craft", "ulysses", "scrivener",
			"typora", "marktext", "zettlr", "ghostwriter", "ia writer",
			"word", "winword.exe", "microsoft word", "excel", "excel.exe",
			"powerpoint", "powerpnt.exe", "outlook.exe",
			"pages", "numbers", "keynote", "wps", "wps office", "金山办公",
			"libreoffice", "openoffice",
			"airtable", "coda", "clickup", "asana", "trello", "monday", "linear",
			"jira", "confluence", "todoist", "things", "omnifocus",
			"reminders", "calendar", "fantastical", "cron", "notion calendar",
			"raycast", "alfred", "launchbar", "spotlight",
		},
		"reading": {
			"acrobat", "acrobat reader", "adobe acrobat", "preview", "skim", "pdf expert",
			"foxit reader", "foxit pdf", "sumatra pdf", "okular", "evince", "zathura",
			"calibre", "kindle", "amazon kindle", "books", "apple books", "ibooks",
			"weread", "微信读书", "duokan", "moon+ reader", "kobo", "readest", "koreader",
			"polar bookshelf", "zotero", "mendeley", "endnote", "papers",
		},
		"social": {
			"twitter", "instagram", "facebook", "reddit", "snapchat", "tiktok",
			"douyin", "抖音", "xiaohongshu", "小红书", "rednote", "weibo", "微博",
			"zhihu", "知乎", "tieba", "贴吧", "bluesky", "mastodon", "threads", "tumblr",
			"pinterest", "linkedin", "v2ex", "quora",
		},
	}
}

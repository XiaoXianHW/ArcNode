package config

// defaultGamingProcessKeywords returns the curated list of process_name
// keywords used to classify gaming activity. Entries are normalized to lower
// case at load time and matched against process_name only (title matching is
// deliberately disabled for gaming to avoid e.g. a Chrome window titled
// "EasyLAN - Minecraft Mod" being classified as gaming).
func defaultGamingProcessKeywords() []string {
	return []string{
		// Launchers / storefronts
		"steam", "steamwebhelper", "steam.exe", "epic games", "epicgameslauncher",
		"battle.net", "battle.net.exe", "blizzard", "origin", "ea app", "eaapp",
		"eadesktop", "ea desktop", "uplay", "ubisoft connect", "upc.exe",
		"gog galaxy", "gog galaxy 2.0", "rockstar games launcher", "rockstargameslauncher",
		"riot client", "riotclientux", "riotclientservices", "bethesda.net launcher",
		"amazon games", "itch.io", "itch", "playnite", "lutris", "heroic games launcher",
		"xbox", "xboxapp", "xbox app", "wegame", "wegameapp", "tencent games",
		"netease launcher", "163launcher", "mihoyo launcher", "hoyoplay",
		"3dm gamelauncher", "qq game", "tlbb", "kdgw.exe", "miyoushe launcher",

		// MOBAs
		"league of legends", "leagueclient", "leagueclient.exe", "leagueclientux",
		"league of legends.exe", "dota 2", "dota2", "dota2.exe",
		"honor of kings", "wzry", "smite", "heroes of the storm", "heroesofthestorm",
		"pokemon unite", "predecessor", "deadlock", "deadlock.exe",
		"vainglory", "arena of valor", "aov",

		// Tactical / hero shooters
		"valorant", "valorant-win64-shipping", "valorant.exe",
		"counter-strike", "counter-strike 2", "cs2", "cs:go", "csgo", "csgo.exe",
		"cs2.exe", "rainbow six", "r6", "rainbowsix", "r6s.exe", "rainbowsix.exe",
		"overwatch", "overwatch.exe", "overwatch 2", "paladins",
		"the finals", "thefinals", "discovery.exe", "splitgate", "splitgate 2",
		"marvel rivals", "marvel-rivals", "concord", "spectre divide", "off the grid",
		"xdefiant", "deadrop", "exoborne",

		// Battle royale / extraction
		"apex legends", "r5apex", "r5apex.exe", "call of duty", "modernwarfare",
		"modernwarfareii", "modernwarfareiii", "warzone", "blackops6", "black ops 6",
		"destiny 2", "destiny2", "halo", "halo infinite", "haloinfinite.exe",
		"escape from tarkov", "eft", "tarkov", "arena breakout",
		"pubg", "tslgame", "pubg.exe", "pubg mobile", "fortnite", "fortniteclient",
		"fortniteclient-win64-shipping", "battlefield", "bf2042", "bf1", "bf4", "bf5",
		"battlefield 2042", "deltaforce", "delta force", "delta-force",
		"once human", "the day before", "marauders", "hunt showdown",
		"naraka bladepoint", "naraka", "naraka.exe",

		// Survival / sandbox
		"rust", "rustclient", "rustclient.exe", "dayz", "dayz.exe", "arma", "arma3",
		"arma reforger", "insurgency", "insurgency sandstorm", "hell let loose",
		"squad", "post scriptum", "minecraft", "minecraftlauncher", "minecraft launcher",
		"minecraft.windows", "lunar client", "badlion client", "prism launcher",
		"terraria", "starbound", "valheim", "valheim.exe", "the forest", "theforest.exe",
		"sons of the forest", "subnautica", "subnautica zero", "raft", "no man's sky",
		"nms.exe", "7 days to die", "7daystodie", "satisfactory", "factorio",
		"core keeper", "vintage story", "palworld", "palworld.exe", "enshrouded",
		"stranded deep", "green hell", "the long dark", "icarus", "v rising",
		"conan exiles", "ark", "ark survival evolved", "ark survival ascended",
		"ark.exe", "rust 2", "soulmask", "ravenfield", "project zomboid",
		"projectzomboid64", "dyson sphere program", "shapez", "shapez 2",
		"mindustry", "oxygen not included", "rimworld", "rimworldwin64",
		"dwarf fortress", "kenshi", "wayward", "unturned",

		// Survival horror / co-op
		"phasmophobia", "lethal company", "content warning", "the outlast trials",
		"deep rock galactic", "deep rock galactic survivor", "remnant", "remnant 2",
		"left 4 dead", "l4d", "back 4 blood", "vermintide", "vermintide 2",
		"darktide", "warhammer 40k darktide", "warhammer 40k", "warhammer 40000",
		"helldivers", "helldivers 2", "helldivers2.exe",
		"among us", "amongus", "fall guys", "fallguys", "fallguys_client_game.exe",
		"human fall flat", "human: fall flat", "gang beasts", "stumble guys",
		"stick fight", "stick fight: the game", "goose goose duck", "town of salem",
		"phantasy star online 2", "pso2",

		// Open-world / single player AAA
		"the witcher", "witcher3", "witcher 3", "cyberpunk2077", "cyberpunk",
		"cyberpunk2077.exe", "elden ring", "eldenring", "eldenring.exe",
		"dark souls", "darksouls", "dark souls iii", "darksoulsiii", "sekiro",
		"sekiro.exe", "bloodborne", "armored core vi", "armoredcore6", "ac6",
		"lies of p", "monster hunter", "monsterhunterrise", "monster hunter rise",
		"monsterhunterworld", "mhw", "monster hunter world", "monster hunter wilds",
		"dragon's dogma", "dragonsdogma2", "dragon's dogma 2", "hogwarts legacy",
		"hogwartslegacy", "rdr2", "red dead redemption", "rdr2.exe", "gta5", "gtav",
		"gtaiv", "gta_sa", "gta v", "gta vi", "grand theft auto",
		"baldur's gate 3", "bg3", "baldursgate3", "starfield", "starfield.exe",
		"skyrim", "skyrimse", "skyrim special edition", "fallout4", "fallout 76",
		"fallout76", "fallout new vegas", "obsidian.exe",
		"persona", "persona 3", "persona3", "persona 4 golden", "persona4",
		"persona 5", "persona5", "persona 5 royal", "yakuza", "like a dragon",
		"ryu ga gotoku", "judgment", "lost judgment",
		"silent hill", "silent hill 2", "resident evil", "biohazard", "re4",
		"re village", "resident evil 4", "resident evil village", "resident evil 9",
		"diablo iv", "diablo4", "diablo 4", "diablo iii", "diablo immortal",
		"path of exile", "poe", "pathofexile.exe", "path of exile 2", "poe2",
		"last epoch", "grim dawn", "torchlight", "wolcen", "v rising",
		"god of war", "god of war ragnarok", "horizon zero dawn", "horizon forbidden west",
		"the last of us", "tlou", "uncharted", "spider-man", "marvels spider-man",
		"alan wake", "alan wake 2", "control", "death stranding", "metro exodus",
		"dishonored", "deathloop", "prey", "doom eternal", "doometernal",
		"the callisto protocol", "atomic heart", "rage 2", "wolfenstein",
		"days gone", "ghost of tsushima",

		// MMOs
		"world of warcraft", "wow.exe", "wow classic", "final fantasy", "ffxiv",
		"ffxiv_dx11", "ffxiv_dx11.exe", "ff14", "lost ark", "lostark.exe",
		"guild wars 2", "gw2", "gw2-64.exe", "elder scrolls online", "eso",
		"eso.exe", "black desert", "blackdesert64.exe", "blade & soul", "tera",
		"new world", "newworld.exe", "throne and liberty", "tnl", "swtor",
		"ragnarok", "ragnarok online", "ro", "albion online", "albion-online",
		"runescape", "runelite", "old school runescape", "osrs", "wakfu", "dofus",
		"eve online", "eveonline.exe", "mortal online 2", "ashes of creation",
		"path of exile 2", "lord of the rings online", "lotro", "rift",
		"perfect world", "perfectworld", "jx3", "jx online 3", "jianwang3",
		"剑网3", "tian xia", "tx3",

		// HoYoverse / Chinese / Korean / Japanese live-service
		"genshin impact", "genshinimpact.exe", "yuanshen.exe", "honkai",
		"honkai impact", "honkai impact 3rd", "honkaistarrail.exe",
		"honkai: star rail", "starrail.exe", "zenless zone zero", "zenlesszonezero",
		"zenlesszonezero.exe", "wuthering waves", "wuwa", "wuthering",
		"punishing gray raven", "pgr", "tower of fantasy", "snowbreak",
		"snowbreak: containment zone", "girls' frontline 2", "girlsfrontline",
		"path to nowhere", "reverse:1999", "reverse 1999",
		"arknights", "uma musume", "blue archive", "nikke", "goddess of victory",
		"counter:side", "epic seven", "epic7", "azur lane",
		"granblue fantasy relink", "granblue fantasy versus", "blue protocol",
		"phantasy star online 2 ngs", "pso2ngs", "lost epic", "soulworker",

		// Sports / racing
		"forza horizon", "forza motorsport", "forzahorizon5", "forzahorizon4",
		"forzahorizon3", "f1 2024", "f12024", "f12023", "f1 24", "ea sports wrc",
		"gran turismo", "gran turismo 7", "gt7", "asphalt", "asphalt 9", "asphalt 8",
		"need for speed", "nfs", "nfs unbound", "burnout", "burnout paradise",
		"the crew", "the crew 2", "the crew motorfest", "wreckfest",
		"dirt rally", "dirt 5", "ride 5", "moto gp", "automobilista",
		"car mechanic simulator", "carx street", "beamng", "beamng.drive",
		"fifa", "ea sports fc", "fc25", "fc24", "fc 25", "nba 2k", "nba2k24",
		"nba2k25", "madden", "madden nfl", "pes", "efootball", "rocket league",
		"rocketleague", "tony hawk", "skate.", "skater xl", "session",
		"top spin 2k25", "wwe 2k", "ufc 5",

		// Strategy / 4X / RTS / grand strategy
		"civilization", "civilizationvi", "civ7", "civ6", "civilization vii",
		"civilization vi", "humankind", "old world", "millennia",
		"stellaris", "europa universalis", "europauniversalis4", "eu4",
		"hearts of iron", "hearts of iron iv", "hoi4", "crusader kings",
		"crusader kings iii", "ck3", "imperator rome", "victoria 3",
		"age of empires", "aoe4", "aoe2", "aoe 4", "starcraft", "starcraft ii",
		"sc2", "warcraft iii", "warcraft3", "warcraft iii reforged", "wcr",
		"company of heroes", "company of heroes 3", "coh3", "total war",
		"total war warhammer", "totalwarhammer", "total war pharaoh", "tww3",
		"warhammer", "songs of conquest", "northgard", "they are billions",
		"world in conflict", "rise of nations", "wargame", "steel division 2",

		// Roguelike / roguelite / indie hits
		"hollow knight", "hollow knight silksong", "celeste", "hades", "hades 2",
		"stardew valley", "stardewvalley", "cuphead", "ori", "ori and the will of the wisps",
		"ori and the blind forest", "undertale", "deltarune", "katana zero",
		"limbo", "inside", "vampire survivors", "vampire-survivors", "balatro",
		"balatro.exe", "slay the spire", "slay the spire 2", "into the breach",
		"ftl", "dead cells", "rogue legacy", "rogue legacy 2", "loop hero",
		"risk of rain", "risk of rain 2", "risk of rain returns", "noita",
		"enter the gungeon", "exit the gungeon", "the binding of isaac",
		"isaac.exe", "isaac repentance", "spelunky", "spelunky 2",
		"hyper light drifter", "hyper light breaker", "katana zero", "neon white",
		"katana zero", "scarlet hollow", "disco elysium", "tunic", "manor lords",
		"animal well", "lorelei and the laser eyes", "thank goodness you're here",
		"unicorn overlord", "pacific drive", "spiritfarer", "outer wilds",
		"chained echoes", "live a live", "octopath traveler", "bloodstained",
		"sea of stars", "darkest dungeon", "darkest dungeon ii", "darkest dungeon 2",
		"shogun showdown",

		// Card / auto-battler / deckbuilder
		"hearthstone", "hearthstone.exe", "legends of runeterra", "lor",
		"marvel snap", "magic the gathering arena", "mtg arena", "mtga",
		"yu-gi-oh master duel", "yu-gi-oh", "yugioh", "shadowverse",
		"teamfight tactics", "tft", "auto chess", "dota underlords",

		// Multiplayer / family / party
		"animal crossing", "splatoon", "splatoon 3", "mario kart", "mario kart 8",
		"smash bros", "super smash bros", "ssbu", "super mario", "the legend of zelda",
		"zelda", "tears of the kingdom", "totk", "metroid", "pokemon",
		"pokemon scarlet", "pokemon violet", "pokemon legends",
		"luigi's mansion", "kirby",

		// Other / VR / emulators / chinese tencent
		"vrchat", "vrchat.exe", "chillout vr", "neos vr", "rec room", "recroom",
		"beat saber", "blade and sorcery", "boneworks", "bonelab", "h3vr",
		"steamvr", "vrmonitor", "oculus", "meta quest",
		"yuzu", "ryujinx", "citra", "dolphin", "pcsx2", "rpcs3", "ppsspp",
		"retroarch", "duckstation", "shadps4", "mednafen", "snes9x", "fceux",
		"openmsx", "mame", "nestopia", "visualboyadvance", "vba", "mgba",
		"flycast", "redream", "dosbox", "scummvm", "openra",
		"crossfire", "crossfire hd", "tlbb", "lol mobile", "wild rift",
		"clash of clans", "clash royale", "brawl stars",

	}
}

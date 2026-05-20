package category

import (
	"regexp"
	"strings"
)

var (
	extOnce       = regexp.MustCompile(`(?i)\.([a-z0-9_+#-]{1,12})(?:[\s"'\)\]\}:,;]|$)`)
	parensOnce    = regexp.MustCompile(`(?i)\(([^()]*)\)`)
	dashSegment   = regexp.MustCompile(`\s[-—–]\s`)
	codeAnnotated = regexp.MustCompile(`(?i)(branch|main|master|develop)[\s:]+`)
)

var extToLang = map[string]string{
	"go": "Go", "mod": "Go",
	"rs": "Rust",
	"ts": "TypeScript", "tsx": "TypeScript", "mts": "TypeScript", "cts": "TypeScript",
	"js": "JavaScript", "jsx": "JavaScript", "mjs": "JavaScript", "cjs": "JavaScript",
	"py": "Python", "pyi": "Python", "ipynb": "Python",
	"java": "Java",
	"kt": "Kotlin", "kts": "Kotlin",
	"swift": "Swift",
	"m": "Objective-C", "mm": "Objective-C",
	"c": "C", "h": "C",
	"cpp": "C++", "cxx": "C++", "cc": "C++", "hpp": "C++", "hh": "C++", "hxx": "C++",
	"cs": "C#",
	"fs": "F#", "fsx": "F#", "fsi": "F#",
	"vb": "VB.NET",
	"rb": "Ruby", "erb": "Ruby",
	"php": "PHP", "phtml": "PHP",
	"pl": "Perl", "pm": "Perl",
	"lua": "Lua",
	"dart": "Dart",
	"r": "R", "rmd": "R",
	"jl": "Julia",
	"scala": "Scala", "sbt": "Scala", "sc": "Scala",
	"clj": "Clojure", "cljs": "Clojure", "cljc": "Clojure", "edn": "Clojure",
	"ex": "Elixir", "exs": "Elixir",
	"erl": "Erlang", "hrl": "Erlang",
	"hs": "Haskell", "lhs": "Haskell", "cabal": "Haskell",
	"elm": "Elm",
	"ml": "OCaml", "mli": "OCaml",
	"ada": "Ada", "adb": "Ada", "ads": "Ada",
	"d": "D",
	"nim": "Nim", "nims": "Nim",
	"v": "V",
	"zig": "Zig",
	"cr": "Crystal",
	"groovy": "Groovy", "gradle": "Groovy",
	"sol": "Solidity",
	"sh": "Shell", "bash": "Shell", "zsh": "Shell", "fish": "Shell",
	"ps1": "PowerShell", "psm1": "PowerShell", "psd1": "PowerShell",
	"bat": "Batch", "cmd": "Batch",
	"sql": "SQL",
	"html": "HTML", "htm": "HTML",
	"css": "CSS", "scss": "Sass", "sass": "Sass", "less": "Less", "styl": "Stylus",
	"vue": "Vue", "svelte": "Svelte", "astro": "Astro",
	"md": "Markdown", "mdx": "Markdown",
	"json": "JSON", "json5": "JSON", "jsonc": "JSON",
	"yaml": "YAML", "yml": "YAML",
	"toml": "TOML",
	"xml": "XML", "xsl": "XML", "xsd": "XML", "wsdl": "XML",
	"dockerfile": "Docker",
	"tf": "Terraform", "tfvars": "Terraform", "hcl": "Terraform",
	"proto": "Protobuf",
	"graphql": "GraphQL", "gql": "GraphQL",
	"tex": "LaTeX", "ltx": "LaTeX", "bib": "LaTeX",
	"asm": "Assembly", "s": "Assembly", "nasm": "Assembly",
	"f": "Fortran", "f90": "Fortran", "f95": "Fortran", "for": "Fortran",
	"cob": "COBOL", "cbl": "COBOL",
	"matlab": "MATLAB", "mat": "MATLAB",
	"vhd": "VHDL", "vhdl": "VHDL",
	"sv": "SystemVerilog", "vhdl_2008": "VHDL",
}

var ideContextLang = map[string]string{
	"goland":      "Go",
	"clion":       "C++",
	"webstorm":    "JavaScript",
	"phpstorm":    "PHP",
	"pycharm":     "Python",
	"rubymine":    "Ruby",
	"rustrover":   "Rust",
	"datagrip":    "SQL",
	"android studio": "Kotlin",
	"xcode":       "Swift",
	"swift playgrounds": "Swift",
	"appcode":     "Objective-C",
	"rider":       "C#",
	"visual studio": "C#",
	"jupyter":     "Python",
	"rstudio":     "R",
	"matlab":      "MATLAB",
	"octave":      "MATLAB",
	"spyder":      "Python",
	"unity":       "C#",
	"unrealeditor": "C++",
	"ue5editor":   "C++",
	"ue4editor":   "C++",
}

var titleContains = []struct {
	keyword string
	lang    string
}{
	{"package.json", "JavaScript"},
	{"tsconfig", "TypeScript"},
	{"cargo.toml", "Rust"},
	{"go.mod", "Go"},
	{"pyproject.toml", "Python"},
	{"requirements.txt", "Python"},
	{"pubspec.yaml", "Dart"},
	{"gemfile", "Ruby"},
	{"composer.json", "PHP"},
	{"build.gradle", "Kotlin"},
	{"podfile", "Swift"},
	{"makefile", "C"},
	{"cmakelists", "C++"},
	{"dockerfile", "Docker"},
}

// DetectLanguage tries to figure out the programming language being edited from
// the window title (which most IDEs format as "filename.ext - project - ide").
// Falls back to nothing if the IDE doesn't surface a filename.
func DetectLanguage(processName, windowTitle string) string {
	title := strings.ToLower(windowTitle)
	process := strings.ToLower(processName)

	if lang := matchByExtension(title); lang != "" {
		return lang
	}
	for _, tc := range titleContains {
		if strings.Contains(title, tc.keyword) {
			return tc.lang
		}
	}
	// Pull the part inside the last parentheses (Xcode/Android Studio sometimes
	// puts the active file there).
	if matches := parensOnce.FindAllStringSubmatch(title, -1); len(matches) > 0 {
		if lang := matchByExtension(matches[len(matches)-1][1]); lang != "" {
			return lang
		}
	}
	// Some editors prefix branch/file info; try each dash-separated segment.
	if dashSegment.MatchString(title) {
		for _, seg := range dashSegment.Split(title, -1) {
			seg = strings.TrimSpace(codeAnnotated.ReplaceAllString(seg, ""))
			if lang := matchByExtension(seg); lang != "" {
				return lang
			}
		}
	}
	for k, lang := range ideContextLang {
		if strings.Contains(process, k) || strings.Contains(title, k) {
			return lang
		}
	}
	return ""
}

func matchByExtension(s string) string {
	matches := extOnce.FindAllStringSubmatch(s, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		ext := strings.ToLower(matches[i][1])
		if lang, ok := extToLang[ext]; ok {
			return lang
		}
	}
	return ""
}

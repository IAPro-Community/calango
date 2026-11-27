package pty

import (
	"runtime"
	"strings"
)

// allowedEnv lists the variables a terminal inherits from the daemon. It is an
// allowlist on purpose: the daemon environment may hold CALANGO_VAULT_KEY, provider
// API keys or tokens, and a denylist would miss names nobody thought of. Names are
// compared case-insensitively because Windows environment names are.
var allowedEnv = map[string]bool{
	"PATH": true, "HOME": true, "USER": true, "LOGNAME": true, "SHELL": true,
	"LANG": true, "LANGUAGE": true, "TZ": true, "TMPDIR": true, "TEMP": true, "TMP": true,
	"DISPLAY": true, "WAYLAND_DISPLAY": true, "XDG_RUNTIME_DIR": true, "XDG_CONFIG_HOME": true,
	"XDG_DATA_HOME": true, "XDG_CACHE_HOME": true, "XDG_STATE_HOME": true,
	// Windows.
	"SYSTEMROOT": true, "WINDIR": true, "COMSPEC": true, "PATHEXT": true, "SYSTEMDRIVE": true,
	"USERPROFILE": true, "USERNAME": true, "USERDOMAIN": true, "HOMEDRIVE": true, "HOMEPATH": true,
	"APPDATA": true, "LOCALAPPDATA": true, "PROGRAMDATA": true, "PROGRAMFILES": true,
	"PROGRAMFILES(X86)": true, "PROGRAMW6432": true, "COMMONPROGRAMFILES": true,
	"COMMONPROGRAMFILES(X86)": true, "PUBLIC": true, "ALLUSERSPROFILE": true,
	"NUMBER_OF_PROCESSORS": true, "PROCESSOR_ARCHITECTURE": true, "OS": true, "PSMODULEPATH": true,
}

// FilterEnv keeps only allowlisted variables from environ (KEY=value entries) and sets
// the terminal type. Locale variables LC_* pass; anything named CALANGO_* never does,
// even if it is later added to the allowlist by mistake.
func FilterEnv(environ []string) []string {
	out := make([]string, 0, len(allowedEnv)+2)
	for _, kv := range environ {
		name, _, ok := strings.Cut(kv, "=")
		if !ok || name == "" {
			continue
		}
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "CALANGO_") {
			continue
		}
		if upper == "TERM" || upper == "COLORTERM" {
			continue
		}
		if allowedEnv[upper] || strings.HasPrefix(upper, "LC_") {
			out = append(out, kv)
		}
	}
	if runtime.GOOS != "windows" {
		out = append(out, "TERM=xterm-256color", "COLORTERM=truecolor")
	}
	return out
}

package harness

import "strings"

// safeCLIEnv keeps only platform/runtime variables needed to locate and launch
// a locally authenticated CLI. Credentials and daemon configuration do not
// cross this boundary implicitly; the turn token is injected separately.
// This is not an OS sandbox: the child may still read files available to its user.
func safeCLIEnv(parent []string) []string {
	out := make([]string, 0, len(parent))
	for _, entry := range parent {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		switch strings.ToUpper(name) {
		case "PATH", "HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH",
			"SYSTEMROOT", "WINDIR", "COMSPEC", "PATHEXT", "SYSTEMDRIVE",
			"APPDATA", "LOCALAPPDATA", "PROGRAMDATA", "PROGRAMFILES", "PROGRAMFILES(X86)",
			"TEMP", "TMP", "TMPDIR", "LANG", "LANGUAGE", "LC_ALL", "LC_CTYPE",
			"TERM", "COLORTERM", "TZ", "CODEX_HOME",
			"SSL_CERT_FILE", "SSL_CERT_DIR", "NODE_EXTRA_CA_CERTS",
			"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR",
			"DBUS_SESSION_BUS_ADDRESS":
			out = append(out, entry)
		}
	}
	return out
}

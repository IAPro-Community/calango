package harness

import "github.com/IAPro-Community/IAPro-SDK/runtime"

// safeCLIEnv expresses Calango's host allowlist; the SDK implements filtering.
// This is not an OS sandbox, and credentials are injected separately.
func safeCLIEnv(parent []string) []string {
	return runtime.AllowEnvironment(parent,
		"PATH", "HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH",
		"SYSTEMROOT", "WINDIR", "COMSPEC", "PATHEXT", "SYSTEMDRIVE",
		"APPDATA", "LOCALAPPDATA", "PROGRAMDATA", "PROGRAMFILES", "PROGRAMFILES(X86)",
		"TEMP", "TMP", "TMPDIR", "LANG", "LANGUAGE", "LC_ALL", "LC_CTYPE",
		"TERM", "COLORTERM", "TZ", "CODEX_HOME",
		"SSL_CERT_FILE", "SSL_CERT_DIR", "NODE_EXTRA_CA_CERTS",
		"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR",
		"DBUS_SESSION_BUS_ADDRESS",
	)
}

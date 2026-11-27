# Backup and restore — local Calango node

## Scope and safety

The `CALANGO_DATA_DIR` contains `calango.db`, provider account homes, Maestro installations/configuration, imported skills (`skills/`) and workspace files. It contains private conversations and other user data. Back it up only to encrypted, access-controlled storage.

**Stop the Calango desktop app and daemon before copying or restoring.** SQLite uses WAL; copying a live data directory can omit committed writes. This runbook uses an offline whole-directory snapshot. Keep the archive outside the data directory.

`CALANGO_DATA_DIR` overrides the default `os.UserConfigDir()/calango`:

- Linux: `${XDG_CONFIG_HOME:-$HOME/.config}/calango`
- macOS: `$HOME/Library/Application Support/calango`
- Windows: `%APPDATA%\calango`

The database alone is not enough to recover secrets. Desktop master keys live in the OS keyring; restore the same OS account/keyring. If `CALANGO_VAULT_KEY_FILE` or `CALANGO_VAULT_KEY` is used, preserve that key source separately in encrypted storage and restore the same key version. Never put a plaintext key in the archive. Calango does not back up the person's `~/.codex` login.

## Create a backup

1. Close Calango and confirm its daemon process has exited.
2. Set `DATA_DIR` to the actual `CALANGO_DATA_DIR` (or its platform default), and set `BACKUP` to an encrypted-storage location outside `DATA_DIR`.
3. Archive the whole directory; verify the archive can be listed.

Linux example:

```sh
if [ -n "${CALANGO_DATA_DIR:-}" ]; then
  DATA_DIR="$CALANGO_DATA_DIR"
elif [ -n "${XDG_CONFIG_HOME:-}" ]; then
  DATA_DIR="$XDG_CONFIG_HOME/calango"
else
  DATA_DIR="$HOME/.config/calango"
fi
BACKUP="/secure-backups/calango-$(date -u +%Y%m%dT%H%M%SZ).tar"
tar -C "$DATA_DIR" -cf "$BACKUP" .
tar -tf "$BACKUP" >/dev/null
```

macOS example: set `DATA_DIR` to `${CALANGO_DATA_DIR:-$HOME/Library/Application Support/calango}`, then use the same `tar -C "$DATA_DIR" -cf "$BACKUP" .` and `tar -tf` commands.

Windows PowerShell example:

```powershell
$DataDir = if ($env:CALANGO_DATA_DIR) { $env:CALANGO_DATA_DIR } else { Join-Path $env:APPDATA 'calango' }
$Backup = Join-Path $env:USERPROFILE 'secure-backups\calango-before-upgrade.tar'
New-Item -ItemType Directory -Force (Split-Path $Backup) | Out-Null
tar.exe -c -f $Backup -C $DataDir .
tar.exe -t -f $Backup | Out-Null
```

Store the matching vault key source with the backup using a separate encrypted secret store. A successful archive listing does not prove the key is recoverable; document and test that separately for the deployment.

## Restore and roll back an application update

1. Stop Calango completely. Preserve the current data directory under a new name; do not merge it with the backup.
2. Create a clean `CALANGO_DATA_DIR` and extract the complete archive into it.
3. Restore the matching vault key source and environment configuration before starting Calango.
4. For a failed application update, restore the **pre-update** data directory and its key source before launching the previous binary. Never run an older binary against a database already migrated by a newer binary; migrations have no down path.
5. Start Calango and confirm startup, `/healthz`, and the expected records. Keep the failed directory and backup until recovery is confirmed.

Example extraction after stopping the app and moving the existing directory aside:

```sh
mkdir -m 700 "$DATA_DIR"
tar -C "$DATA_DIR" -xf "$BACKUP"
```

## Verification boundary

`internal/store/backup_restore_test.go` creates a database at the previous migration version, snapshots it while closed, applies the pending migration, restores the snapshot, and verifies the record survives when the current binary reapplies that migration. It does **not** execute a previous released binary or validate a real OS keyring backup. Those platform-specific rollback checks remain release gates.

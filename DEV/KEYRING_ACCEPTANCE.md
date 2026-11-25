# Native keyring acceptance (#55)

Opt-in test with a random credential and dedicated test namespace. Never use a production keyring for destructive testing. In a disposable VM or session with an unlocked keyring:

Windows PowerShell:

```powershell
$env:CALANGO_TEST_OS_KEYRING = '1'
go test ./internal/vault -run '^TestNativeKeyringRoundTrip$' -count=1 -v
Remove-Item Env:CALANGO_TEST_OS_KEYRING
```

macOS/Linux:

```sh
CALANGO_TEST_OS_KEYRING=1 go test ./internal/vault -run '^TestNativeKeyringRoundTrip$' -count=1 -v
```

The test verifies creation, rereading and removal through the native backend without printing the key or using the real account. A SKIP in ordinary CI is not acceptance evidence. Additionally validate in a VM: startup with unlocked keyring, restart preserving the token, no plaintext fallback, explicit failure with locked keyring and recovery with the same user. Record OS, version, commit and sanitized output. For #63, repeat with a snapshot and previous/current binaries.

Do not delete real account credentials. Do not attach keys, tokens, personal paths or the database. Windows and macOS results are mandatory to close #55; cross-builds are not runtime evidence.

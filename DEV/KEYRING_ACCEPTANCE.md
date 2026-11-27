# Aceitação do keyring nativo (#55)

Teste opt-in com credencial aleatória e namespace de teste dedicado. Nunca use o keyring de produção para teste destrutivo. Em VM descartável ou sessão com keyring destravado:

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

O teste verifica criação, releitura e remoção pelo backend nativo sem imprimir a chave nem usar a conta real. SKIP no CI comum não é evidência de aceite. Valide também em VM: start com keyring destravado, restart preservando o token, ausência de fallback em texto claro, falha explícita com keyring travado e recuperação com o mesmo usuário. Registre OS, versão, commit e saída sanitizada. Para #63, repita com snapshot e binários anterior/atual.

Não apague credenciais reais da conta. Não anexe chaves, tokens, caminhos pessoais ou o banco. Resultados em Windows e macOS são obrigatórios para encerrar #55; cross-builds não são evidência de runtime.
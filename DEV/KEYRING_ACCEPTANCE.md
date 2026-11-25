# Homologação do keyring nativo (#55)

Teste opt-in, com credencial aleatória e namespace exclusivo de teste. Nunca usar keyring de produção para ensaio destrutivo. Em VM ou sessão descartável, com o keyring desbloqueado:

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

O teste verifica criação, releitura e remoção via backend nativo sem imprimir a chave ou usar a conta real. Um SKIP na CI comum não equivale a sucesso de homologação. Validar adicionalmente em VM: startup com keyring desbloqueado, restart mantendo token, ausência de fallback em texto claro, falha explícita com keyring bloqueado e recuperação com o mesmo usuário. Registrar sistema, versão, commit e saída sanitizada. Para #63, repetir com snapshot e binários anterior/atual.

Não execute a exclusão de credenciais da conta real. Não anexar chave, token, caminho pessoal nem banco. Os resultados de Windows e macOS são obrigatórios para fechar #55; builds cruzados não são evidência de runtime.

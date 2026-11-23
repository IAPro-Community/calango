# Módulo: vault

Fase 1 · Pacote `internal/vault` · Tabela `vault_items` (só referência)

## Objetivo
Guardar segredos (chaves de modelo, tokens de conector, senhas de sites) no cofre do sistema operacional.

## Regras
- Backends: Windows Credential Manager (DPAPI), macOS Keychain, Linux Secret Service. Na VPS (sem cofre do SO): arquivo criptografado com chave vinda de variável de ambiente ou Docker secret, fora do volume de dados.
- Linux sem keyring destravado: recusar salvar e explicar como destravar. Nunca cair para texto puro.
- API devolve só nome e data; o valor nunca volta.
- O agente usa segredo por referência (`vault://nome`); o valor é resolvido no momento da chamada e mascarado em logs e passos.
- Agente pode pedir um segredo à pessoa (`request_secret`): cartão em que a pessoa preenche; o agente é avisado quando o valor está disponível.

## Aceite
- Nenhum segredo aparece em `run_steps`, logs ou respostas da API (teste de canário).

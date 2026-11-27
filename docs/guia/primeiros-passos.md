# Primeiros passos

## Disponível hoje

O Calango ainda não tem versão publicada. Para usar, você compila a partir do código. O app desktop sobe o processo `calango` sozinho e conversa com ele só pela própria máquina (`127.0.0.1:7317`).

### O que você precisa

- Git.
- Node.js 22.12 ou mais novo.
- Go 1.23 ou mais novo (o build compila o processo `calango`).
- Um modelo para conversar, de um destes jeitos:
  - **Codex:** o [Codex CLI](https://github.com/openai/codex) instalado e com login feito (`codex login`);
  - **Anthropic:** uma chave da API Anthropic.
- Um cofre do sistema operacional desbloqueado (chaveiro do GNOME/KDE no Linux, Keychain no macOS, Gerenciador de Credenciais no Windows). É lá que fica a chave do cofre do Calango.

### Instalar e abrir

```bash
git clone https://github.com/IAPro-Community/calango.git
cd calango
npm ci
npm run desktop:build
npm run desktop:start
```

`desktop:build` compila o processo `calango` em `bin/calango`, o processo principal do Electron e a interface. `desktop:start` abre o app.

> [!NOTE]
> Executado no Linux. No Windows e no macOS os mesmos comandos devem funcionar, mas não foram executados para este guia. O instalador Windows está em validação interna; quando sair, este passo muda.

Se o app não abrir e o terminal mostrar `OS keyring unavailable`, desbloqueie o cofre do sistema e tente de novo. O Calango não guarda segredos em texto puro; sem cofre, ele não sobe.

### Conectar um modelo

Na primeira abertura aparece **Configure seu primeiro agente**. Clique em **Conectar um modelo** e escolha o tipo de conta:

- **Codex:** usa o login do Codex já feito neste computador. Para uma segunda conta Codex, o app mostra o comando de login com uma pasta própria (`CODEX_HOME=... codex login`).
- **Anthropic:** cole a chave. Ela vai para o cofre e nunca volta para a tela.

As contas ficam em **Configurações → Conexões**.

### Criar o primeiro agente e conversar

1. Em **Agentes**, crie um agente, escolha a conta e defina **Quando pedir aprovação**:
   - **Sempre perguntar**;
   - **Só ações sensíveis** (padrão);
   - **Confiar nas ações comuns**.
2. Em **Conversas**, abra uma conversa com o agente e mande a primeira mensagem. Você pode parar uma resposta no meio.
3. Quando o agente quiser fazer algo que precisa de OK, a execução fica em **Aguardando sua aprovação** e o pedido aparece em **Aprovações**. Enviar, apagar, publicar, pagar ou fazer deploy sempre espera você.

### Outras telas

- **Rotinas:** pedidos que o agente roda sozinho, por agenda cron (por exemplo `0 8 * * 1-5`) ou por intervalo (`1h`). Dá para pausar, retomar e rodar uma vez na hora.
- **Uso e custos:** tokens e custo agrupados por agente, rotina ou modelo. Quando o preço do modelo não é conhecido, a linha aparece como custo desconhecido.
- **Auditoria:** verifica a cadeia de registros (cada entrada depende da anterior) e exporta em JSONL ou CSV.
- **Configurações:** contas, cofre de segredos, Orquestrador Maestro ([página própria](maestro.md)) e **Iniciar com o sistema**, que vem desligado.

> [!NOTE]
> **Iniciar com o sistema** foi conferido no Linux, onde cria e remove a entrada de autostart. Windows e macOS ainda não foram validados.

### Conectores MCP

O Calango já roda conectores MCP e segura toda ação de escrita até você aprovar. Por enquanto eles são cadastrados pela API (`/v1/connectors` e `/v1/agents/{id}/tools` em [`api/openapi.yaml`](../../api/openapi.yaml)). Ainda não há tela para isso no app.

## Fases futuras

- Instalador Windows assinado (fase 1, em validação) e pacotes Linux e macOS (v1.0.0).
- Tela de conectores, catálogo com OAuth (v0.7.0) e assistente de criação de agentes por linguagem natural (v0.7.0).
- Mais providers: Claude Code CLI, OpenCode e outros (v1.0.0).

## Veja também

- [Privacidade](privacidade.md): onde ficam os dados.
- [`DEV/BACKUP_RESTORE.md`](../../DEV/BACKUP_RESTORE.md): backup antes de atualizar.

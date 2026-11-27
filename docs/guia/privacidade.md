# Privacidade

O Calango guarda seus dados na sua máquina (ou no seu VPS) e não manda telemetria. Esta página diz o que fica guardado, onde fica e o que sai da máquina.

> [!NOTE]
> Esta página descreve o comportamento do código hoje. Não é a política de privacidade nem os termos de uso, que ainda serão publicados (v0.4.0).

## Onde ficam os dados

Tudo fica na pasta de dados. `CALANGO_DATA_DIR` troca o local; sem ela, o padrão é:

| Sistema | Pasta |
|---|---|
| Linux | `${XDG_CONFIG_HOME:-$HOME/.config}/calango` |
| macOS | `$HOME/Library/Application Support/calango` |
| Windows | `%APPDATA%\calango` |

Dentro dela:

| Item | Conteúdo |
|---|---|
| `calango.db` | banco SQLite: agentes, conversas e mensagens, execuções, rotinas, memórias, aprovações, uso, dispositivos, auditoria e o cofre (criptografado) |
| `accounts/` | uma pasta `CODEX_HOME` para cada conta Codex extra, com o login dela |
| `maestro/` | versões instaladas do Maestro e o login do Codex de cada uma |
| `workspace/` | pasta de trabalho dos agentes |

A pasta é criada só para o seu usuário. O conteúdo é privado: conversas, memórias e arquivos de trabalho.

## Segredos

- Chaves de API, tokens de conectores e o token local da API ficam no **cofre**, criptografados dentro do `calango.db`.
- A chave que abre o cofre fica **fora** do banco:
  - no desktop, no cofre do sistema operacional, no serviço `calango`, conta `vault-master-v1`;
  - no VPS, em `CALANGO_VAULT_KEY_FILE` ou `CALANGO_VAULT_KEY`.
- Sem cofre do sistema e sem essa variável, o Calango não sobe. Ele nunca grava segredo em texto puro.
- Valores do cofre não aparecem na lista e não voltam pela API. O token local só sai pelo comando `calango print-token`, que o app desktop usa para se conectar.

## O que sai da máquina

O Calango não tem telemetria nem envio de erros. As conexões de saída são estas:

| Para onde | O que vai | Quando |
|---|---|---|
| Provider do modelo: API da Anthropic, ou o Codex CLI com o seu login | a mensagem atual e até cerca de 1.500 tokens das suas memórias como contexto; o Codex também pode ler e enviar arquivos da pasta de trabalho | a cada execução de agente |
| `registry.npmjs.org` | consulta de versões e download do Orquestrador Maestro | uma vez por dia e quando você instala ou atualiza |
| Conectores MCP que você cadastrar | o que o agente pedir à ferramenta; ações de escrita só depois da sua aprovação | quando o agente usa a ferramenta |

Outros pontos:
- O que o provider faz com os dados segue os termos dele (Anthropic, ou OpenAI no caso do Codex).
- O Maestro roda com a telemetria dele desligada.
- Conectores locais (stdio) recebem do ambiente só `PATH` e `HOME`; suas outras variáveis de ambiente não vão para eles.

## Quem acessa o nó

- O processo `calango` do desktop escuta só em `127.0.0.1`. A API aceita o token local ou um token de dispositivo pareado, e o navegador só é aceito com a origem do próprio app.
- Cada dispositivo pareado tem escopo próprio e pode ser revogado ([Pareamento](pareamento.md)).
- Num VPS, o acesso de fora deve passar por TLS ([Calango no VPS](vps.md)).

## Memória, auditoria e apagar dados

- **Memórias:** apagar uma memória a tira do uso na hora. A remoção física acontece 30 dias depois, numa limpeza diária.
- **Auditoria:** ações importantes (aprovações, troca de versão do Maestro, entre outras) ficam numa cadeia de registros encadeados por hash. A tela **Auditoria** verifica a cadeia e exporta em JSONL ou CSV.
- **Apagar tudo:**
  1. feche o app e confirme que o processo `calango` terminou;
  2. apague a pasta de dados;
  3. apague a entrada `calango` / `vault-master-v1` do cofre do sistema.

  Seu login pessoal do Codex (`~/.codex`) não é tocado pelo Calango.
- **Backup:** faça com o app fechado e guarde em armazenamento criptografado, com a chave do cofre separada ([`DEV/BACKUP_RESTORE.md`](../../DEV/BACKUP_RESTORE.md)).

## Fases futuras

- Política de privacidade e termos de uso (v0.4.0).
- Relay de notificações para o celular, que só avisa "há novidade", sem o conteúdo (fase 2).
- Replicação entre nós por eventos, entre máquinas suas (fase 3).

## Veja também

- [`DEV/SECURITY.md`](../../DEV/SECURITY.md): modelo de ameaças e defesas.

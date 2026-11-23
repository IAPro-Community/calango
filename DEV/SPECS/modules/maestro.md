# Módulo: maestro

Fase 1 (instalação e vínculo) · Fase 3 (atualização automática com rollback) · Pacote `internal/maestro` · ADR 0009 · Guia: [../ORQUESTRADOR_MAESTRO.md](../ORQUESTRADOR_MAESTRO.md)

## Objetivo
Embutir o Orquestrador Maestro (linha V1, npm `@iapro/orquestrador-maestro-cli`) e acompanhar as atualizações oficiais sem risco para os agentes.

## Interface (Go)
```go
type MaestroManager struct { /* home, npm runner, registry client, store, clock */ }
func (m *MaestroManager) Status(ctx context.Context) (Status, error)
func (m *MaestroManager) Install(ctx context.Context, version string) error   // grava versão nova, verifica, troca ponteiro
func (m *MaestroManager) CheckUpdate(ctx context.Context) (*Release, error)  // consulta registro, só linha V1
func (m *MaestroManager) Rollback(ctx context.Context) error
```

## Regras
- Linha fixada em `^1.1.1`. Versão maior nunca instala sozinha; sempre cartão com aviso de quebra.
- Instalação em `<calango-home>/maestro/versions/<versão>`; ponteiro `current` aponta para a ativa.
- `verify` obrigatório depois de instalar ou atualizar. Falhou → mantém a anterior e registra o motivo.
- Verificação diária do registro npm; falha de rede não gera alerta, só registra.
- Atualização automática: desligada por padrão; ligada por dono. Nunca roda durante um run.
- Um agente só usa o Maestro se o dono vinculou. Desvincular não apaga a instalação.
- Limpeza: manter as 3 últimas versões que passaram no `verify`.
- Toda troca de versão gera `audit_log` com `from`, `to`, `auto` e resultado.
- Tela "Sobre" mostra versão ativa, licença Apache-2.0 e link do repositório.

## API
`GET /v1/maestro`, `POST /v1/maestro/install`, `POST /v1/maestro/update`, `POST /v1/maestro/rollback`, `PATCH /v1/maestro/settings` `{ auto_update: bool }`.

## Eventos
`maestro.update_available`, `maestro.updated`, `maestro.update_failed`.

## Aceite
- Versão nova na V1 com `auto_update` desligado gera cartão e não troca nada.
- Versão que falha no `verify` não substitui a ativa.
- Rollback restaura a versão anterior e o `verify` passa.
- Versão 2.x nunca é instalada sem confirmação explícita.

# Histórico de alterações

Alterações relevantes serão registradas por versão publicada.

## Não publicado

- Variante `maestro` da imagem do nó (`X.Y.Z-maestro`, Node.js 24 LTS e npm, não root), padrão do instalador (`--variant minimal` para a distroless), para o Orquestrador Maestro se instalar e se atualizar no VPS; teste de ponta a ponta em `deploy/test/maestro-vps-e2e.sh`.
- Instalador de VPS em uma linha (`deploy/install.sh`), Compose com Caddy/TLS automático, chave do cofre como secret e `CALANGO_TRUSTED_PROXIES`, subcomando `calango healthcheck` e publicação da imagem multi-arquitetura (amd64/arm64) no GHCR por tag de release.
- Governança orientada a issues, templates de contribuição e gate de CI (pendente de revisão e confirmação administrativa).
- Política operacional em português brasileiro e registro retroativo do trabalho anterior à governança.

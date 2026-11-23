# Verificação

Atualizado: 2026-11-23.

## Núcleo Go (verificado)

Ambiente: Go 1.27.4 linux/amd64, instalado em `~/.local/opt/go`.

| Verificação | Resultado |
|---|---|
| `go vet ./...` | ok, sem saída |
| `go test ./... -count=1` | ok: `TestNewAgentValidates`, `TestChargeRespectsBudget`, `TestSchedulerDueOnInterval`, `TestSchedulerRejectsBadRoutines` |
| `gofmt -l .` | ok, sem arquivos |
| `go build ./cmd/calango` + subida em `127.0.0.1:7399` | ok |
| `GET /healthz` | 200, corpo `{"status":"ok"}` |
| `POST /healthz` | 405 (rota restrita a GET) |
| Processo de teste encerrado | ok |

Não verificado: Docker (imagem), CI, Windows, macOS, Flutter, Electron, Tailscale, Maestro.

## Docker (bloqueado por permissão)

Ambiente: Docker 29.1.3, daemon ativo, usuário `bolzan` no grupo `docker` (adicionado em 2026-11-22).

| Verificação | Resultado |
|---|---|
| `docker build -t calango-site .` (Dockerfile da raiz) | ok, imagem `nginxinc/nginx-unprivileged:1.27-alpine` |
| `nginx -t` dentro do container do site | ok |
| `GET /healthz` do site | 200 |
| `GET /` do site | 200, `text/html` |
| Cabeçalhos CSP, `X-Frame-Options: DENY`, `nosniff` | presentes |
| `docker build -f deploy/Dockerfile.node -t calango/calango .` | ok, 13.3 MB |
| `GET /healthz` do nó (`calango/calango`, usuário nonroot) | 200, `{"status":"ok"}` |
| `docker run` com rede bridge | falha: `operation not supported` ao criar veth neste ambiente |
| Build e run com `--network host` | ok (ambas as imagens) |
| `hadolint` nos Dockerfiles | não executado (não instalado) |

Observação: a rede bridge do Docker não funciona nesta máquina (erro de veth). Os testes de run usaram `--network host`. Em máquinas normais, `docker run -p` deve funcionar; isso ainda não foi verificado aqui.

## Contrato e tokens

| Verificação | Resultado |
|---|---|
| `api/openapi.yaml` é YAML válido e todos os `$ref` resolvem (15 rotas, 11 esquemas) | ok (PyYAML) |
| Validador OpenAPI completo | não executado |
| `packages/tokens/tokens.json` é JSON válido | ok |
| Links de `DEV/INDEX.md` e `ROADMAP.md` para ADRs e specs existem | ok |

## Site (`site/index.html`)

Servido em `http://localhost:4377`, navegador embutido.

| Verificação | Resultado |
|---|---|
| Console sem erros | ok |
| Imagens da logo carregam | ok |
| Demo: 5 cenas com o estado esperado | ok, conferido pelo DOM |
| Texto da notificação legível no modo escuro | corrigido |
| Contraste medido por script (claro e escuro), transições desligadas | falhas de texto corrigidas; restam falsos positivos (gradiente do celular e nav com `oklab`, que o script não interpreta) |
| Título da seção ADE no modo escuro | estava invisível; corrigido e conferido em captura |
| Marcador dos títulos (`.sweep`) em claro e escuro | legível depois da revelação |
| Overflow horizontal em 375 px | não conclusivo: a emulação ficou com 516 px de largura de layout nesta aba |
| Revisão visual completa em claro e escuro, todas as seções | parcial: hero (claro), ADE (escuro) conferidos |

## Orquestrador Maestro

| Verificação | Resultado |
|---|---|
| Versão estável da linha V1 identificada (1.1.1, release de 2026-10-18) | ok, por releases do GitHub |
| Licença Apache-2.0 confirmada | ok (arquivo `LICENSE`) |
| Instalação via npm e `verify` no Calango | não executado |

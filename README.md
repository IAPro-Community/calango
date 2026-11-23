# Calango

Assistente pessoal que roda no seu computador, continua na sua VPS e responde no seu celular. Rotinas, aprovações, navegador, ADE para programar projetos e canais com pessoas e agentes.

> **Em construção.** Estamos construindo a V1 em público. Veja o [TODO da V1](DEV/TODO_V1.md) e [como contribuir](CONTRIBUTING.md).

[![Licença: AGPL-3.0](https://img.shields.io/badge/licen%C3%A7a-AGPL--3.0-E8A600)](LICENSE)

## Ver a página do projeto com Docker

```bash
docker build -t calango-site .
docker run --rm -p 8080:8080 calango-site
```

Abra <http://localhost:8080>.

## Rodar o núcleo (Go 1.22+)

```bash
go vet ./...
go test ./...
go run ./cmd/calango          # sobe em 127.0.0.1:7317
curl http://127.0.0.1:7317/healthz
```

## Imagem do nó (para VPS)

```bash
docker build -f deploy/Dockerfile.node -t calango/calango .
```

Exemplo de compose com TLS automático: [deploy/docker-compose.yml](deploy/docker-compose.yml).

## Estrutura

| Pasta | Conteúdo |
|---|---|
| `cmd/calango` | entrada do servidor |
| `internal/` | núcleo: agentes, agendador, API |
| `api/` | contrato OpenAPI (fonte dos SDKs) |
| `packages/tokens` | design tokens (fonte de CSS e tema Flutter) |
| `site/` | página do projeto |
| `deploy/` | Dockerfile do nó, compose, Caddy |
| `DEV/` | documentação de produto e engenharia |

## Documentação

- [AGENTS.md](AGENTS.md): ponto de entrada para quem vai programar (humano ou agente).
- [DEV/INDEX.md](DEV/INDEX.md): mapa de toda a documentação.
- [DEV/PLANO.md](DEV/PLANO.md): panorama completo do produto.

## Licença

Código aberto sob [GNU AGPL-3.0](LICENSE). Licença comercial disponível para quem não pode cumprir a AGPL: [COMMERCIAL.md](COMMERCIAL.md). Contribuições seguem o [CLA](CLA.md). Nome e logo: [TRADEMARKS.md](TRADEMARKS.md).

O Calango usa o [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro) (Apache-2.0) como processo de trabalho dos agentes. Crédito e avisos em [NOTICE](NOTICE).

## Segurança

Veja [SECURITY.md](SECURITY.md). Não abra issue pública para vulnerabilidade.

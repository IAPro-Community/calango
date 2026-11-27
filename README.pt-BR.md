<p align="center">
  <img src="site/assets/logo-192.png" width="112" alt="Logo do Calango" />
</p>

<h1 align="center">Calango</h1>

<p align="center">
  <strong>Seu assistente pessoal. Roda no seu computador, continua na sua VPS e responde no seu celular.</strong>
</p>

<p align="center">
  Rotinas · aprovações · navegador · ADE para programar projetos · canais com pessoas e agentes
</p>

<p align="center">
  <a href="LICENSE"><img alt="Licença: AGPL-3.0" src="https://img.shields.io/badge/licen%C3%A7a-AGPL--3.0-E8A600" /></a>
  <img alt="Status: em construção" src="https://img.shields.io/badge/status-em%20constru%C3%A7%C3%A3o-2D7A22" />
  <img alt="Go 1.25+" src="https://img.shields.io/badge/go-1.25%2B-00ADD8" />
</p>

<p align="center">
  <a href="README.md">English</a> · <a href="README.es.md">Español</a> · <b>Português (BR)</b>
</p>

---

> [!NOTE]
> **Em construção.** Estamos construindo a V1 em público. Veja o [TODO da V1](DEV/TODO_V1.md) e [como contribuir](CONTRIBUTING.md).

## Por que Calango

- **Lembra, executa e guarda o que você pede.** Trabalha no computador que você já usa.
- **Nada importante acontece sem a sua ok.** Enviar, apagar, publicar, pagar ou fazer deploy sempre espera uma aprovação.
- **Seus dados ficam com você.** Cada nó guarda o próprio banco. Segredos nunca chegam aos clientes.
- **Controle de qualquer lugar.** Use o aplicativo de desktop, o celular ou os dois. A VPS continua trabalhando quando o computador dorme.

## Início rápido

### Ver a página do projeto (Docker)

```bash
docker build -t calango-site .
docker run --rm -p 8080:8080 calango-site
```

Depois abra <http://localhost:8080>.

### Rodar o núcleo (Go 1.25+)

```bash
go vet ./...
go test ./...
go run ./cmd/calango          # sobe em 127.0.0.1:7317
curl http://127.0.0.1:7317/healthz
```

### Gerar a imagem do nó (para VPS)

```bash
docker build -f deploy/Dockerfile.node -t calango/calango .
```

Exemplo de compose com TLS automático: [deploy/docker-compose.yml](deploy/docker-compose.yml).

## Estrutura do repositório

| Pasta | Conteúdo |
|---|---|
| `cmd/calango` | entrada do servidor |
| `internal/` | núcleo: agentes, agendador, API |
| `api/` | contrato OpenAPI (fonte dos SDKs) |
| `packages/tokens` | design tokens (fonte de CSS e do tema Flutter) |
| `packages/i18n` | catálogo trilíngue (TypeScript e ARB para Flutter) |
| `site/` | página do projeto |
| `deploy/` | Dockerfile do nó, compose, Caddy |
| `DEV/` | documentação de produto e engenharia |

## Gerar e verificar tokens e traduções

Os arquivos em `packages/tokens/generated` e `packages/i18n/generated` são derivados das fontes JSON. Regenere-os após editar a fonte e confira o drift com:

```bash
node scripts/generate-tokens.mjs
node scripts/generate-i18n.mjs
node scripts/generate-tokens.mjs --check
node scripts/generate-i18n.mjs --check
```

## Documentação

- [docs/guia](docs/guia/README.md): guia de uso em pt-BR (primeiros passos, pareamento, VPS, Maestro e privacidade).
- [AGENTS.md](AGENTS.md): ponto de entrada para quem vai programar (humano ou agente).
- [DEV/INDEX.md](DEV/INDEX.md): mapa de toda a documentação.

## Idioma do projeto

O repositório é em **inglês**. Issues, pull requests, mensagens de commit, comentários de código e documentação devem ser escritos em inglês. Veja [CONTRIBUTING.md](CONTRIBUTING.md#language).

## Licença

Código aberto sob [GNU AGPL-3.0](LICENSE). Licença comercial para quem não pode cumprir a AGPL: [COMMERCIAL.md](COMMERCIAL.md). Contribuições seguem o [CLA](CLA.md). Nome e logo: [TRADEMARKS.md](TRADEMARKS.md).

O Calango usa o [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro) (Apache-2.0) como processo de trabalho dos agentes. Crédito e avisos em [NOTICE](NOTICE).

## Segurança

Veja [SECURITY.md](SECURITY.md). Não abra issue pública para vulnerabilidade.

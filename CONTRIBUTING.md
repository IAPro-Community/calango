# Como contribuir com o Calango

Obrigado por querer ajudar. O Calango é um assistente pessoal que roda no seu computador e na sua VPS. Estamos construindo a **V1** em público.

Antes de tudo, leia:
- [LICENSE](LICENSE) (AGPL-3.0)
- [CLA.md](CLA.md): você precisa aceitar o acordo de contribuidor no seu primeiro PR
- [TRADEMARKS.md](TRADEMARKS.md): regras de uso do nome e da logo
- [AGENTS.md](AGENTS.md): regras para contribuir com código, inclusive com agentes de IA
- [DEV/TODO_V1.md](DEV/TODO_V1.md): o que falta para a V1

## Por onde começar

1. Escolha uma tarefa aberta no [TODO da V1](DEV/TODO_V1.md) ou no [roadmap](DEV/SPECS/ROADMAP.md).
2. Se a tarefa tem um identificador (ex.: `F1-T03`), use-o no título do PR.
3. Para mudança grande, abra uma issue antes com a proposta. Decisões de arquitetura viram ADR em `DEV/ADR/`.
4. Ainda não sabe por onde ir? Os itens marcados como "bom primeiro issue" são pequenos e bem definidos.

## Ambiente de desenvolvimento

Requisitos: Go 1.22 ou superior. Node.js 20.19 ou superior para o Orquestrador Maestro.

```bash
go vet ./...
go test ./...
go run ./cmd/calango          # sobe em 127.0.0.1:7317
```

Site de apresentação:

```bash
docker build -t calango-site .
docker run --rm -p 8080:8080 calango-site
# abra http://localhost:8080
```

## Regras do código

- Siga [DEV/CONVENTIONS.md](DEV/CONVENTIONS.md).
- Mudança de API começa em `api/openapi.yaml`.
- Mudança de banco é uma migração nova. Migrações commitadas não são editadas.
- Ação sensível nunca pode ser executada sem aprovação do dono (ver [DEV/SECURITY.md](DEV/SECURITY.md)).
- Não envie segredos, chaves, dados pessoais ou logs com conteúdo de conversa.
- Não copie código de outros projetos. Veja [ADR 0006](DEV/ADR/0006-codigo-original.md). A exceção do Orquestrador Maestro está no [ADR 0009](DEV/ADR/0009-orquestrador-maestro.md).

## Pull requests

- Um assunto por PR.
- Testes para regra de domínio.
- Descreva o que mudou, por quê e como verificar.
- Atualize a documentação em `DEV/` quando a mudança afetar escopo, arquitetura ou API.
- Aceite o CLA quando o bot pedir.

## Código de conduta

Esperamos respeito e colaboração. Quem não seguir isso pode ser removido do projeto. Relate problemas ao contato em [COMMERCIAL.md](COMMERCIAL.md).

## Segurança

Não abra issue pública para vulnerabilidade. Veja [SECURITY.md](SECURITY.md).

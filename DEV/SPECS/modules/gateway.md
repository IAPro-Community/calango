# Módulo: gateway

Fase 1 · Pacote `internal/gateway` · Tabela `usage` · ADR 0007

## Objetivo
Ponto único de toda chamada de modelo: orçamento antes, uso depois.

## Interface (Go)
```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req Request, stream func(Chunk)) (Usage, error)
}
type Gateway struct { /* store, providers, prices, clock */ }
func (g *Gateway) Call(ctx context.Context, who Caller, req Request, stream func(Chunk)) (Usage, error)
```
`Caller` = agente, rotina (opcional), projeto (opcional), nó.

## Regras
1. Antes: soma do mês do agente + estimativa da chamada ≤ limite. Senão `budget_exceeded` e nada sai.
2. Depois: grava `usage` com custo em micro-unidades (inteiro, sem float).
3. Emite `usage.threshold` em 80% e 100%.
4. Tabela de preços por modelo versionada no código; preço desconhecido = custo 0 e alerta no log.
5. Nenhum pacote fora do gateway importa SDK de provider.

## Aceite
- Teste: agente com limite 100 e uso 95 não chama provider para requisição estimada em 10.
- Teste: custo gravado bate com tokens × preço.

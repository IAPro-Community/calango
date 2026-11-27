# Orquestrador Maestro

O [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro) (Apache-2.0, da IAPro Community) organiza agentes de código com regras, roteamento de skills, contexto compacto, verificação e passagem de bastão. O Calango instala e mantém o Maestro para você.

## Disponível hoje

### O que você precisa

- Node.js 20.19 ou mais novo e o `npm` no PATH (o Maestro é instalado pelo npm).
- Acesso a `registry.npmjs.org`.

### Instalar

Em **Configurações → Orquestrador Maestro**, clique em **Instalar versão V1**. O Calango:
1. baixa a versão estável da linha V1 numa pasta própria, `<pasta de dados>/maestro/versions/<versão>`, nunca na sua pasta pessoal;
2. roda a verificação do Maestro e só ativa a versão se ela passar;
3. desliga a telemetria do Maestro;
4. registra a troca de versão na auditoria.

A tela mostra a **Versão ativa** e o crédito com a licença.

### Vincular a um agente

O Maestro é opcional por agente. Na tela **Agentes**, use **Vincular ao Maestro** ou **Desvincular do Maestro**.

O Maestro usa um login do Codex **separado** do seu. O perfil fica em `maestro/versions/<versão>/home/.codex`, um por versão instalada. Seu login pessoal do Codex não é copiado. Depois de instalar, faça o login nesse perfil:

```bash
CODEX_HOME="<pasta de dados>/maestro/versions/<versão>/home/.codex" codex login
```

A pasta de dados está em [Privacidade](privacidade.md#onde-ficam-os-dados).

### Atualizar e reverter

- **Atualizar agora** instala uma versão nova da linha V1 e verifica antes de ativar. Se a verificação falhar, a versão atual continua.
- **Reverter versão** volta para a versão anterior guardada.
- **Instalar atualizações automaticamente** vem ligado. O Calango consulta o npm uma vez por dia e nunca troca de versão com uma execução em andamento. Versões fora da linha V1 (2.x em diante) não são instaladas.

## Limitações conhecidas

- A consulta ao npm falha hoje: o aviso de versão nova do próprio npm se mistura com a resposta, e numa instalação nova a pasta do Maestro ainda não existe. O log mostra `maestro automatic update failed` ou `maestro registry check failed`. Na prática:
  - o Maestro não se instala sozinho na primeira abertura;
  - o aviso de versão nova não aparece, e **Atualizar agora** pode falhar.

  Instale pelo botão **Instalar versão V1**, que funciona.
- O perfil do Codex do Maestro muda a cada versão instalada; depois de uma atualização, pode ser preciso fazer o login de novo.
- A imagem Docker do VPS não traz Node.js; o Maestro não funciona lá.

## Fases futuras

- Validação da atualização automática no VPS (fase 3, v0.4.0).
- Importar skills do Claude Code e do Codex (fase 4, v0.5.0).

## Veja também

- [`DEV/ORQUESTRADOR_MAESTRO.md`](../../DEV/ORQUESTRADOR_MAESTRO.md) e [ADR 0014](../../DEV/ADR/0014-maestro-isolamento.md).

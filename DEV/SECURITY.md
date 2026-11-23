# Segurança

## Ativos

Chaves de modelo e de conectores, arquivos do PC, sessões do navegador, código dos projetos, conversas e memória, controle da tela do PC.

## Modelo de ameaças

| Ameaça | Defesa | Onde |
|---|---|---|
| Alguém na rede acessa o nó | toda rota exige token de dispositivo, inclusive na LAN; Tailscale; TLS na VPS | `pairing-auth` |
| Token de dispositivo roubado | token com escopo e validade, revogação por dispositivo, lista de dispositivos visível | `pairing-auth` |
| Agente executa algo destrutivo | aprovação obrigatória por tipo de ação; pasta de trabalho restrita; Docker na VPS | `approvals`, `tools-mcp` |
| Prompt injection (web, e-mail, arquivo) | conteúdo externo entra marcado como dado; ferramentas de escrita sempre passam por aprovação; o agente não pode aprovar a si mesmo | `harness`, `approvals` |
| Vazamento de chaves | cofre do sistema operacional; o agente usa por referência, nunca vê o valor; máscara em logs e traces | `vault` |
| Gasto descontrolado | orçamento por agente e por rotina, checado antes de cada chamada | `gateway` |
| Controle remoto da tela abusado | desligado por padrão; ligar só no próprio PC; aviso visível enquanto ativo; auditoria; encerra com um clique | `computer-use` |
| Licença forjada no nó VPS | chave assinada Ed25519, validada offline; rebaixa para o grátis, nunca derruba | `licensing-billing` |
| Push vazando conteúdo | relay só carrega "tem novidade" + id opaco | `push` |
| Atualização maliciosa | instaladores assinados; imagem Docker com digest; canal de update com assinatura | `desktop-electron`, `cloud` |

## Regras

1. Nenhuma rota sem autenticação, exceto `/healthz` e o início do pareamento.
2. O agente nunca decide uma aprovação. Só pessoas aprovam.
3. Auditoria append-only de: aprovações, ações remotas, controle de tela, mudanças de permissão, dispositivos pareados.
4. Segredo nunca é retornado pela API depois de salvo.
5. Saída de rede por agente pode ser restrita a uma lista de destinos (fase posterior).

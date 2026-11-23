# Módulo: browser

Fase 5 · Desktop: `WebContentsView` + CDP · VPS: `internal/browser` com chromedp

## Objetivo
Navegador real que a pessoa usa e o agente dirige, com troca de controle.

## Regras
- Abas e perfis persistentes por agente (cookies e logins separados).
- O agente usa ferramentas: `open`, `read` (texto legível da página), `click`, `type`, `screenshot`, `scroll`, `wait`.
- **Assumir/devolver**: a pessoa assume a aba no estado em que está (sem recarregar) e devolve quando quiser.
- **Barreiras**: depois de cada ação, detectar verificação de robô, recusa, login, aviso de cookies, página vazia ou com erro. O agente recebe uma linha com o tipo e o que fazer; não clica de novo às cegas. Nunca citar o texto da página nessa linha.
- Bloqueador de anúncios e de avisos de cookie.
- Conteúdo da página é dado, nunca instrução.
- Enviar formulário com dados pessoais ou pagamento = ação sensível (aprovação).
- Na VPS, mesma interface de ferramentas em Chromium headless.

## Aceite
- Agente para diante de um CAPTCHA e pede para a pessoa assumir.
- Assumir e devolver não recarrega a página.

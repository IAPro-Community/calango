# Módulo: ade (ambiente de desenvolvimento com agentes)

Fase 4 · Pacotes `internal/ade`, `internal/pty` · Tabelas `projects`, `worktrees`, `tasks`

## Objetivo
Programar projetos separados com agentes, no PC ou na VPS.

## Regras
- Projeto = repositório git + nó + agentes + conectores + memória (`DEV/` do projeto, se existir).
- Cada tarefa abre um **worktree** num branch próprio. Vários agentes em paralelo sem conflito.
- Terminal: PTY no `calango` (`creack/pty`; ConPTY no Windows), transmitido por WebSocket (`/v1/pty/{id}`); xterm.js no desktop.
- Editor Monaco, árvore de arquivos, diff por worktree.
- Comandos de shell passam pela lista segura; fora dela, aprovação.
- Servidor de preview (porta detectada) abre no navegador interno.
- GitHub/GitLab: criar branch, abrir PR, ler status do CI.
- Ao terminar a tarefa: oferecer PR, manter ou descartar o worktree.
- Importar skills de Claude Code (`.claude/`) e Codex (`.codex/`).

## Aceite
- Duas tarefas no mesmo projeto rodam em paralelo em worktrees diferentes e geram dois PRs.
- Projeto na VPS editável pelo desktop com o mesmo editor e terminal.

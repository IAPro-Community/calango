# Instalador Windows interno (não assinado)

Pacote para testes internos do desktop Electron com o daemon `calango` embutido (issue #116, subtarefa de #39). **Não é distribuição pública**: o instalador não é assinado e não tem atualização automática.

## Gerar

Requisitos: Go 1.23+, Node 22.12+ e Windows. Em Linux o empacotamento para antes do instalador NSIS porque o electron-builder precisa do wine; a pasta `apps/desktop/release/win-unpacked` sai normalmente.

```bash
npm ci
npm run desktop:package:win
```

O comando compila `bin/calango.exe` (`scripts/build-daemon.mjs`: `CGO_ENABLED=0`, `-trimpath`, sem build id, então o mesmo commit gera o mesmo binário), o main/preload e o renderer, e gera `apps/desktop/release/Calango-<versão>-interno-nao-assinado-x64.exe`. A configuração está em `apps/desktop/electron-builder.yml`; a versão vem de `apps/desktop/package.json`.

A CI (job `desktop-windows-package`, runner `windows-latest`) gera o instalador e o `.sha256`, instala em modo silencioso, sobe o daemon instalado com chave de cofre descartável, confere `/healthz` e que o log não contém o token, desinstala e publica o artefato `calango-windows-interno-nao-assinado` por 14 dias.

## Conteúdo do pacote

| Caminho no app instalado | Conteúdo |
|---|---|
| `Calango.exe` | shell Electron |
| `resources/app.asar` | main, preload e renderer compilados (sem `node_modules`; o renderer já vem empacotado pelo Vite) |
| `resources/bin/calango.exe` | daemon Go |
| `resources/legal/` | `LICENSE`, `NOTICE` e `third_party/` |
| `LICENSE.electron.txt`, `LICENSES.chromium.html` | licenças do Electron e do Chromium |

## Testar em máquina ou VM Windows descartável

1. Baixe o artefato da CI e confira o checksum: `Get-FileHash .\Calango-*.exe -Algorithm SHA256` deve bater com o `.sha256`.
2. Rode o instalador. O SmartScreen avisa que o editor é desconhecido: use "Mais informações → Executar assim mesmo" só nessa máquina de teste. Não desative o SmartScreen nem outros controles do Windows.
3. A instalação é por usuário (padrão `%LocalAppData%\Programs\Calango`) e permite trocar a pasta.
4. Abra o Calango. Sem `CALANGO_VAULT_KEY_FILE`, o daemon cria a chave mestra no Gerenciador de Credenciais do Windows (validação do keyring em #55).
5. Verifique:
   - `curl http://127.0.0.1:7317/healthz` responde `{"status":"ok"}`;
   - a janela mostra o daemon como "running";
   - reinício: finalize o daemon no Gerenciador de Tarefas; o app sobe outro e `/healthz` volta a responder. O Windows não diferencia maiúsculas, então `Calango.exe` (Electron) e `calango.exe` (daemon) parecem o mesmo processo: finalize o que está em `resources\bin\calango.exe`;
   - log do daemon em `%AppData%\Calango Desktop\logs\daemon.log`, sem token nem valores do cofre;
   - dados em `%AppData%\calango` (`calango.db`).
6. Saia pelo menu da bandeja ("Sair"), desinstale por "Aplicativos instalados" e confirme que a pasta do programa sumiu. O desinstalador mantém os dados em `%AppData%\calango` e a credencial `calango:vault-master-v1`; apague os dois à mão para zerar a máquina.

Registre na issue: versão do Windows, hash do instalador, resultado de cada passo e o trecho relevante do `daemon.log`.

## Limitações

- Sem assinatura de código: SmartScreen e antivírus podem avisar ou bloquear. Assinatura e atualização assinada ficam em #39 e dependem de certificado.
- Sem atualização automática; cada versão é instalada por cima.
- A desinstalação mantém os dados em `%AppData%\calango` e a chave no Gerenciador de Credenciais (sem exclusão de dados pessoais ainda; ver #46).
- Só x64. Ícone gerado a partir de `site/assets/logo-512.png`, sem versão simplificada para tamanhos pequenos (#49).
- `appId` provisório (`io.github.iapro-community.calango`) até a decisão de domínio (#44).
- Se o processo do Electron for encerrado à força, o `calango.exe` filho pode continuar rodando; a limpeza de processos descendentes é tratada em #113.

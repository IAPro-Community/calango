# Pacotes do desktop para Linux e macOS

Pacotes do desktop Electron com o daemon `calango` embutido (issue #40, F1-T25b). O Linux gera `.deb` e AppImage x64, validados em contêiner limpo. O macOS tem o DMG configurado, mas **depende de credenciais Apple**: assinatura e notarização estão desligadas e nada para macOS foi gerado nem testado em Mac real.

Nenhum pacote é assinado nem tem atualização automática. A CI não publica nenhum deles.

## Linux

### Gerar

Requisitos: Go 1.25+, Node 22.12+ e Linux x64. O electron-builder baixa o Electron, o `appimagetool` e o `fpm` das fontes oficiais dele (cache em `~/.cache/electron` e `~/.cache/electron-builder`); nada é instalado no sistema.

```bash
npm ci
npm run desktop:package:linux
```

O comando compila `bin/linux-x64/calango` (`scripts/build-daemon.mjs`: `CGO_ENABLED=0`, `-trimpath`, sem build id), o main/preload e o renderer, gera `apps/desktop/release/Calango-<versão>-linux-amd64.deb` e `Calango-<versão>-linux-x86_64.AppImage` e, no fim, roda `scripts/verify-desktop-package.mjs` sobre `release/linux-unpacked/resources`. A configuração está em `apps/desktop/electron-builder.yml`; a versão vem de `apps/desktop/package.json`.

### Conteúdo do `.deb`

| Caminho | Conteúdo |
|---|---|
| `/opt/Calango/calango-desktop` | shell Electron (não se chama `calango` para não confundir com o daemon) |
| `/usr/bin/calango-desktop` | link via `update-alternatives` |
| `/opt/Calango/resources/app.asar` | main, preload e renderer compilados, sem `node_modules` |
| `/opt/Calango/resources/bin/calango` | daemon Go (executável) |
| `/opt/Calango/resources/legal/` | `LICENSE`, `NOTICE` e `third_party/` integrais (inclui `iapro-sdk` e `agenticos`) |
| `/opt/Calango/LICENSE.electron.txt`, `LICENSES.chromium.html` | licenças do Electron e do Chromium |
| `/usr/share/applications/calango-desktop.desktop` e ícone 512 px | entrada de menu |

Pacote `calango-desktop`, licença `AGPL-3.0`. Dependências: as padrão do electron-builder mais `libgbm1` e `libasound2t64 | libasound2 (>= 1.0.17)`. Sem as duas o Electron 45 não abre em instalação sem Recommends; sem a versão na ALSA, o apt do Ubuntu 24.04 escolhe `liboss4-salsa-asound2` e o app quebra com `undefined symbol: snd_device_name_get_hint`. Os dois problemas apareceram no teste em contêiner.

O AppImage leva o mesmo conteúdo em `resources/` (conferido com `--appimage-extract` + `verify-desktop-package.mjs`). O `.desktop` do AppImage usa `--no-sandbox`, padrão do electron-builder.

### Gate de licenças

- `scripts/verify-desktop-package.mjs --resources <pasta resources>` falha se `LICENSE`, `NOTICE` ou qualquer arquivo de `third_party/` faltar ou divergir byte a byte do repositório, ou se `bin/calango` faltar ou não for executável. Roda no fim de `package:linux` e `package:mac`.
- `scripts/verify-desktop-package.test.mjs` (em `npm run contracts:check`) testa o gate com pacotes falsos (completo, sem o `NOTICE` do IAPro-SDK, `LICENSE` alterada, sem daemon, daemon sem `+x`) e confere que o `electron-builder.yml` copia os arquivos legais e o daemon em Windows, Linux e macOS, e que o macOS segue sem assinatura nem notarização.

### Validação em contêiner

```bash
npm run desktop:package:linux:test            # debian:12 e ubuntu:24.04
bash scripts/test-linux-package.sh <atual.deb> <anterior.deb>
```

Sem `.deb` anterior, o script reempacota o atual com a versão `<versão>~anterior` para exercitar a atualização pelo dpkg. Em cada imagem ele:

1. instala a versão anterior sem Recommends, sobe o daemon instalado como usuário comum (chave de cofre aleatória, pasta temporária) e para;
2. atualiza para o `.deb` atual e confere versão, arquivos, link em `/usr/bin`, `Exec` do `.desktop`, ausência de arquivo gravável por qualquer usuário, licenças integrais e `ldd` sem bibliotecas faltando;
3. sobe o daemon sobre os dados da versão anterior: `/healthz` = `{"status":"ok"}`, `print-token` devolve o token e os logs não o contêm;
4. abre o app instalado em Xvfb e confere que ele sobe `resources/bin/calango`, que `/healthz` responde e que `~/.config/Calango Desktop/logs/daemon.log` existe sem o token;
5. faz `apt-get purge` e confere que `/opt/Calango`, o link, a alternativa, o `.desktop` e o ícone sumiram e que os dados do usuário ficaram.

O job `desktop-linux-package` da CI roda `desktop:package:linux` e esse teste, sem enviar artefatos. Ele não entrou no `quality-gate`.

## macOS (pendente de credenciais)

`apps/desktop/electron-builder.yml` já define DMG x64 e arm64, `hardenedRuntime`, daemon `bin/darwin-<arch>/calango` em `Contents/Resources/bin/calango` e os arquivos legais. Para não assinar por engano, `identity: null` e `notarize: false`; o artefato se chama `Calango-<versão>-mac-<arch>-nao-assinado.dmg`.

`npm run desktop:package:mac` só gera o DMG em macOS. No Linux, `electron-builder --mac dir --x64 --arm64` montou os dois `Calango.app`, e `verify-desktop-package.mjs` passou nos dois, com o daemon Mach-O da arquitetura certa. DMG, assinatura, notarização, instalação e atualização em Mac real continuam pendentes.

Para fechar o macOS da #40:

1. Conta Apple Developer com certificado **Developer ID Application** e credencial de notarização (API key do App Store Connect ou Apple ID com senha de app e Team ID), guardados só no ambiente protegido de release, nunca em PR de fork.
2. Trocar `identity: null` e `notarize: false` pela configuração assinada, com entitlements do hardened runtime para o Electron e para o daemon embutido (que também precisa ser assinado).
3. Gerar em runner macOS, conferir com `codesign --verify --deep --strict`, `spctl -a -vv` e `xcrun stapler validate`, instalar e atualizar em Mac limpo (Intel e Apple Silicon).
4. Atualizar o teste do gate (`identity`/`notarize`) junto com a mudança.

## Limitações

- Sem assinatura (Linux e macOS) e sem atualização automática; atualizar é instalar o pacote novo por cima.
- Só x64 no Linux. arm64 fica para depois.
- A remoção mantém os dados (`~/.config/calango`, `~/.config/Calango Desktop`) e a chave no keyring; a exclusão de dados é a #46. O keyring real do Linux não entra no teste em contêiner (ver #55).
- `appId` (`io.github.iapro-community.calango`) e `maintainer` do `.deb` são provisórios até a decisão de domínio (#44).
- O teste em contêiner precisa de Docker e baixa imagens e pacotes do Debian/Ubuntu.

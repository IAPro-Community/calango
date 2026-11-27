# Calango no VPS

A ideia é o nó rodar num VPS para o agente continuar trabalhando com o computador desligado. Os clientes (desktop e celular) viram controles remotos.

## Situação hoje

> [!WARNING]
> **Exemplo, parcialmente validado.** A imagem compila e sobe, mas o desktop ainda não se conecta a um nó remoto e o celular ainda não existe. Use só para testar. `deploy/docker-compose.yml` ainda **não** funciona como está (veja [Limitações conhecidas](#limitações-conhecidas)).

O que já dá para fazer:
- rodar o nó em Docker com dados persistentes;
- falar com ele pela API, usando o token local ou dispositivos pareados ([Pareamento](pareamento.md));
- usar contas Anthropic (chave de API).

## Passo a passo validado

Executado no Linux com Docker, em rede bridge com porta publicada só em `127.0.0.1`.

### 1. Gerar a imagem

Na raiz do repositório:

```bash
docker build -f deploy/Dockerfile.node -t calango/calango .
```

### 2. Criar a chave do cofre

No contêiner não há cofre do sistema. Sem chave, o nó para com `OS keyring unavailable`. Gere uma chave de 32 bytes num arquivo que só o usuário do contêiner (uid 65532) consiga ler:

```bash
sudo install -d -m 700 /etc/calango
head -c 32 /dev/urandom | base64 | sudo tee /etc/calango/vault.key >/dev/null
sudo chown 65532:65532 /etc/calango/vault.key
sudo chmod 400 /etc/calango/vault.key
```

Guarde uma cópia dessa chave fora do VPS, em armazenamento criptografado. **Sem ela, os segredos do cofre não voltam.** Nunca coloque a chave no backup do volume de dados.

### 3. Preparar o volume de dados

O nó grava em `CALANGO_DATA_DIR`. Sem essa variável, ele grava dentro do contêiner e os dados somem quando o contêiner é recriado. O volume novo nasce com dono `root`, então ajuste o dono uma vez:

```bash
docker volume create calango-data
docker run --rm -v calango-data:/data alpine chown 65532:65532 /data
```

### 4. Subir o nó

```bash
docker run -d --name calango --restart unless-stopped \
  -e CALANGO_DATA_DIR=/data \
  -e CALANGO_VAULT_KEY_FILE=/run/secrets/calango_vault_key \
  -v /etc/calango/vault.key:/run/secrets/calango_vault_key:ro \
  -v calango-data:/data \
  -p 127.0.0.1:7317:7317 \
  calango/calango
```

Confira:

```bash
curl -s http://127.0.0.1:7317/healthz
docker exec calango calango print-token
```

O primeiro comando responde `{"status":"ok"}`. O segundo imprime o token local; trate como senha. Recriar o contêiner com o mesmo volume e a mesma chave mantém dados e token.

### 5. Acesso de fora: só com TLS

A porta 7317 nunca deve ficar aberta sem TLS. Dois caminhos:

- **Proxy reverso com certificado automático** (Caddy ou Traefik). Exemplo em `deploy/Caddyfile`. Defina `CALANGO_TRUSTED_PROXIES` com o endereço do proxy; sem isso, todos os clientes atrás do proxy dividem o mesmo limite de tentativas de pareamento.
- **Rede privada** (WireGuard ou Tailscale), sem porta pública.

O acesso externo por proxy e por rede privada ainda não foi validado.

## Limitações conhecidas

- `deploy/docker-compose.yml` não define a chave do cofre nem `CALANGO_DATA_DIR`. Como está, o nó não sobe; com a chave, sobe, mas os dados ficam fora do volume. Use o passo a passo acima até o exemplo ser corrigido.
- A imagem não traz Node.js nem o Codex CLI. Contas Codex e o Orquestrador Maestro não funcionam no VPS; o log mostra `maestro automatic update failed` uma vez por dia, o que é esperado nessa imagem.
- Backup e restauração do volume Docker ainda não foram validados. O procedimento para o nó local está em [`DEV/BACKUP_RESTORE.md`](../../DEV/BACKUP_RESTORE.md); pare o contêiner antes de copiar o volume.

## Fases futuras

- Imagem multi-arquitetura e instalador para VPS (fase 3, v0.4.0).
- Desktop e celular conectados ao nó remoto, execução PC/VPS/auto e replicação entre nós (fase 3).
- Senha de administrador no primeiro acesso e pareamento por QR Code (fases 2 e 3).
- Hospedagem gerenciada para quem não quer manter um VPS (plano pago, a definir).

## Veja também

- [`DEV/DEPLOYMENT.md`](../../DEV/DEPLOYMENT.md): modelo de implantação e requisitos de segurança.

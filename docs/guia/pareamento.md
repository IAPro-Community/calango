# Pareamento de dispositivos

Parear é dar a outro dispositivo um token próprio para falar com o seu nó Calango, com escopo limitado e que pode ser revogado sozinho.

## Disponível hoje

O pareamento já existe na API do nó. Ainda **não** há tela de pareamento no app desktop nem app de celular. Hoje ele serve para testes e integrações.

### Como funciona

- Um dispositivo com escopo `full` gera um **código de pareamento**: 8 caracteres, uso único, vale 5 minutos.
- O novo dispositivo troca o código por um **token de dispositivo** em `POST /v1/pair`. O token aparece só nessa resposta; o nó guarda apenas o hash.
- A rota de pareamento aceita no máximo 5 tentativas por minuto por origem.
- Escopos:

| Escopo | Pode |
|---|---|
| `full` | tudo, inclusive gerar códigos e gerenciar dispositivos |
| `approve_only` | listar e decidir aprovações |
| `read` | ler agentes e memórias |

### Passo a passo (nó local, Linux)

Com o app aberto (ou o processo `calango` rodando), pegue o token local. O comando só imprime o token, sem registrar em log:

```bash
TOKEN=$(./bin/calango print-token)
```

Gere um código com o escopo que o dispositivo vai ter:

```bash
curl -s -X POST http://127.0.0.1:7317/v1/pairing-codes \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"scope":"approve_only"}'
```

No dispositivo novo, troque o código pelo token:

```bash
curl -s -X POST http://127.0.0.1:7317/v1/pair \
  -H 'Content-Type: application/json' \
  -d '{"code":"<código>","device_name":"Meu notebook","platform":"web"}'
```

`platform` aceita `desktop`, `ios`, `android` ou `web`. Guarde o `token` da resposta num lugar seguro: ele não aparece de novo.

Liste e revogue dispositivos com o token local (ou de um dispositivo `full`):

```bash
curl -s http://127.0.0.1:7317/v1/devices -H "Authorization: Bearer $TOKEN"
curl -s -X DELETE http://127.0.0.1:7317/v1/devices/<device_id> -H "Authorization: Bearer $TOKEN"
```

Depois de revogado, o dispositivo recebe `401` na próxima chamada. Um código já usado também recebe `401`.

> [!WARNING]
> O nó local escuta só em `127.0.0.1`, e o app desktop não aceita outro endereço. Para parear um dispositivo de outra máquina, o nó precisa estar num VPS atrás de TLS ([Calango no VPS](vps.md)). Nunca exponha a porta 7317 sem TLS.

### Trocar o token local

`POST /v1/auth/rotate-token` troca o token local na hora; o antigo para de funcionar. O novo aparece só na resposta. Se ela se perder, ninguém tem token e não há rota que o devolva: o acesso precisa ser restaurado à mão no nó.

## Fases futuras

- Pareamento por QR Code e app de celular para iOS e Android (fase 2, v0.3.0).
- Tela de dispositivos no app desktop.
- Transporte remoto entre dispositivos sem abrir porta pública (fase 2; decisão de transporte pendente).
- Senha de administrador no primeiro acesso ao nó do VPS (fases 2 e 3).

## Veja também

- [`DEV/SPECS/modules/pairing-auth.md`](../../DEV/SPECS/modules/pairing-auth.md): especificação do módulo.

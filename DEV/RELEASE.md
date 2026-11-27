# Como publicar uma release do Calango

A `main` só recebe mudanças por PR de release com origem na `develop`. A revisão humana independente é obrigatória nessa promoção. O fluxo diário das issues termina na `develop`, conforme o [guia de contribuição](../CONTRIBUTING.md).

## 1. Preparar a versão na develop

- Confirme o aceite da entrega e os critérios do milestone. Registre as evidências e o resultado da validação.
- Atualize o [changelog](../CHANGELOG.md) com mudanças, compatibilidade, instruções de atualização e limitações.
- Escolha uma versão SemVer. Use `vX.Y.Z` para uma versão estável ou um sufixo como `vX.Y.Z-rc.1` para uma pré-release.

## 2. Promover para main

1. Abra um PR de release com **base `main`** e **compare `develop`**. Informe a versão, o resumo, as evidências de aceite e os riscos conhecidos.
2. Aguarde o `quality-gate` verde, resolva as conversas e obtenha aprovação humana independente após o último push.
3. O mantenedor autorizado integra com **merge commit**. Copie o SHA desse commit na `main` para identificar a versão que será publicada.

## 3. Criar a tag e publicar no GitHub

O mantenedor autorizado abre [Releases → Draft a new release](https://github.com/IAPro-Community/calango/releases/new), preenche título e notas e seleciona a tag da versão. Para criar a tag pela interface, confira se o destino é o commit aprovado na `main`. Se a interface oferecer somente a ponta da branch e ela já tiver avançado, use a CLI para indicar o SHA exato. Marque **Set as a pre-release** para uma versão de teste.

Exemplo equivalente pela CLI para uma release estável. Substitua a versão e o SHA pelos valores aprovados e prepare as notas em `notas-release.md`.

```bash
gh release create vX.Y.Z --repo IAPro-Community/calango \
  --target SHA_DO_MERGE_NA_MAIN --title "Calango vX.Y.Z" \
  --notes-file notas-release.md
```

Para uma pré-release pela CLI, use a tag com sufixo e acrescente `--prerelease`. Use uma tag nova para cada versão e preserve tags já publicadas.

## 4. Verificar a entrega publicada

- Confirme que a tag aponta para o SHA aprovado na `main` e que as notas e os artefatos previstos estão disponíveis na release.
- A publicação da tag dispara o workflow [release-image](../.github/workflows/release-image.yml) quando ele está presente na revisão publicada. Confira a execução em **Actions** e as duas variantes no GHCR, `X.Y.Z` e `X.Y.Z-maestro`. Consulte o [guia de implantação](DEPLOYMENT.md) para validar os artefatos.
- Registre a URL da release, o SHA, o resultado do workflow e eventuais pendências na issue ou no PR de release.

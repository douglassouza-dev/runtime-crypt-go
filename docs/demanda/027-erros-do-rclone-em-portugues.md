# 027 — Erros do rclone em português e cabeçalho do seletor

- Estado: Aberta
- Risco: Médio · uso
- Onde:
  - `internal/core/montagem.go`: `motivoFalhaMontagem`, `mensagemFalhaMontagem`;
  - `internal/core/executar.go:erroComMotivoDoRclone`;
  - `internal/core/gerenciador.go:CriarRemoto`;
  - `internal/core/oauth.go`;
  - `internal/gui/seletor_pasta.go:novaListaPastas`.
- Depende de: 018, 026

## Contexto

O texto em inglês que o rclone escreve no stderr chega à tela sem tradução:
- `Não destrancou: {última linha do stderr}` no card;
- `Falha ao montar …:` seguido de até 20 linhas do stderr no diálogo;
- o erro do seletor de pasta (`erroComMotivoDoRclone`);
- `Não deu para configurar o {provedor}: {saída do config create}` no wizard;
- `Autorização não concluída: …` no OAuth.

Além disso, o topo do seletor de pasta mostra o nome interno `{nome}_base:/…`.

O app não grava log em arquivo. No Windows, com `-H windowsgui`, o que vai para o `log` se perde.

## O que muda

- Uma função de mapeamento (`core.ClassificarSaidaRclone`) leva a saída do rclone a um conjunto fixo de frases da tela:
  - `senha errada`;
  - `sem conexão com o {provedor}`;
  - `autorização expirou`;
  - `a pasta não existe no {provedor}`;
  - qualquer outra coisa: `o rclone falhou, detalhes no log`.
- O texto original vai só para o log. O app passa a gravar o log em `{pasta do app}/runtimecrypto.log`, que troca para `.log.1` ao passar de 1 MB. Senhas não entram no log.
- O seletor de pasta mostra `{Provedor} /{caminho}` no topo, por exemplo `Google Drive /Backup/cofre`. `{nome}_base:` não aparece.

## O que fica de fora

- Traduzir mensagens que o próprio app escreve em português.
- Tela para ver o log.
- Diferenciar mais casos além dos cinco.

## Pronto quando

- [ ] Teste de tabela de `ClassificarSaidaRclone` com amostras do stderr do rclone 1.60. As capturadas no box ficam marcadas `real` e as escritas à mão (sem como reproduzir aqui) ficam marcadas `suposta`.
- [ ] Teste: falha de montagem com stderr em inglês → card `Não destrancou: {frase}` e diálogo sem o texto em inglês; o texto original está no log.
- [ ] Teste: erro de listagem no seletor e de `config create` no wizard passam pela mesma função.
- [ ] Teste: o topo do seletor é `{Provedor} /` e, dentro de uma pasta, `{Provedor} /a/b`; `_base:` não aparece.
- [ ] Teste: o log é gravado no arquivo e troca de arquivo ao passar do limite.
- [ ] `rg -n '"Saida do rclone' internal` não encontra nada.

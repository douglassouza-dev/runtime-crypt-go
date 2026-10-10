# 027 — Erros do rclone em português e cabeçalho do seletor

- Estado: Aberta
- Risco: Médio · uso
- Onde:
  - `internal/core/montagem.go`: `motivoFalhaMontagem`, `mensagemFalhaMontagem`;
  - `internal/core/executar.go:erroComMotivoDoRclone`;
  - `internal/core/gerenciador.go:CriarRemoto`;
  - `internal/core/oauth.go`;
  - `internal/gui/seletor_pasta.go:novaListaPastas`;
  - `internal/plataforma/*.go:VerificarWinfsp` (driver de montagem);
  - `internal/core/log_arquivo.go`, `main.go` (log).
- Depende de: 018, 026

## Contexto

O texto em inglês que o rclone escreve no stderr chega à tela sem tradução:
- `Não destrancou: {última linha do stderr}` no card;
- `Falha ao montar …:` seguido de até 20 linhas do stderr no diálogo;
- o erro do seletor de pasta (`erroComMotivoDoRclone`);
- `Não deu para configurar o {provedor}: {saída do config create}` no wizard;
- `Autorização não concluída: …` no OAuth.

Além disso, o topo do seletor de pasta mostra o nome interno `{nome}_base:/…`.

Sem o driver de montagem (WinFsp, FUSE, macFUSE), o card mostra a mensagem genérica do rclone (ex.: `cannot find winfsp`).

O app não grava log em arquivo. No Windows, com `-H windowsgui`, o que vai para o `log` se perde.

## O que muda

- Uma função de mapeamento (`core.ClassificarSaidaRclone`) leva a saída do rclone a um conjunto fixo de frases da tela:
  - `senha errada`;
  - `sem conexão com o {provedor}`;
  - `autorização expirou`;
  - `a pasta não existe no {provedor}`;
  - qualquer outra coisa: `o rclone falhou, detalhes no log`.
- O texto original vai só para o log.
- Falta do driver de montagem tem frase própria (aprovada pela UI), no lugar de `Não destrancou: …`:
  - Windows: `Não montou: falta instalar o WinFsp.` e o botão `Baixar WinFsp`, que abre https://winfsp.dev/rel/;
  - Linux: `Não montou: falta instalar o FUSE.` (só o texto);
  - macOS: `Não montou: falta instalar o macFUSE.` e o botão `Baixar macFUSE` (aprovado pela UI), que abre https://macfuse.github.io/.
  - O diálogo de erro ao destrancar mostra a mesma frase.
  - Detecção em duas camadas: antes de iniciar o `rclone mount`, `plataforma.VerificarWinfsp` (Windows: DLL/registro/pasta do WinFsp; Linux: `/dev/fuse` e `fusermount3` ou `fusermount` no PATH; macOS: `macfuse.fs` ou `fuse-t.fs`); se ela disser que está instalado e o rclone falhar mesmo assim, o texto do rclone (`cannot find winfsp`, `cannot find FUSE`, `fuse device not found`, `"fusermount3": executable file not found`) leva à mesma frase.
- Log: `runtimecrypto.log` fica na mesma pasta do `vaults.json` (o `DiretorioApp` do gerenciador) e troca para `.log.1` ao passar de 1 MB. Senhas não entram no log. Se essa pasta não aceita escrita, o app segue sem arquivo de log: não trava e não tenta outra pasta.
  - Hoje essa pasta é a do executável (`obterDiretorioApp`), a mesma do `vaults.json`. Levar as duas para a pasta de configuração do usuário é outra demanda (precisa migrar o `vaults.json`).
- O seletor de pasta mostra `{Provedor} /{caminho}` no topo, por exemplo `Google Drive /Backup/cofre`. `{nome}_base:` não aparece.

## O que fica de fora

- Traduzir mensagens que o próprio app escreve em português.
- Tela para ver o log.
- Diferenciar mais casos além dos listados.
- Instalar o driver pelo app.

## Pronto quando

- [ ] Teste de tabela de `ClassificarSaidaRclone` com amostras do stderr do rclone 1.60. As capturadas no box ficam marcadas `real` e as escritas à mão (sem como reproduzir aqui) ficam marcadas `suposta`.
- [ ] Teste: falha de montagem com stderr em inglês → card `Não destrancou: {frase}` e diálogo sem o texto em inglês; o texto original está no log.
- [ ] Teste: erro de listagem no seletor e de `config create` no wizard passam pela mesma função.
- [ ] Teste: o topo do seletor é `{Provedor} /` e, dentro de uma pasta, `{Provedor} /a/b`; `_base:` não aparece.
- [ ] Teste: o log é gravado no arquivo e troca de arquivo ao passar do limite.
- [ ] Teste: sem o driver, a montagem é recusada sem iniciar o rclone e o card mostra `Não montou: falta instalar o {driver}.`; com o stderr do rclone sem driver (amostra real do Linux), a mesma frase.
- [ ] Teste: frase e botão por sistema (Windows e macOS com botão, Linux sem); o botão abre o site oficial.
- [ ] Teste: o log fica ao lado do `vaults.json`; com a pasta sem escrita, `AbrirLog` devolve erro e o destino do log não muda.
- [ ] `rg -n '"Saida do rclone' internal` não encontra nada.

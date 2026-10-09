# 003 — Falha de montagem só aparece depois de 45 s e sem motivo

- Estado: Aberta
- Risco: Alto — estabilidade
- Onde: `internal/core/montagem.go:MontarUnidade`
- Depende de: 001

## Contexto

`MontarUnidade` faz `cmd.Stdout = nil` e `cmd.Stderr = nil`. A saída do rclone vai para lugar nenhum.

O laço de espera testa `cmd.ProcessState != nil && cmd.ProcessState.Exited()`. `ProcessState` só é preenchido por `Wait()`, e `MontarUnidade` nunca chama `Wait`. Então, se o rclone sai na hora (flag inválida, remoto inexistente, WinFsp ausente, senha errada de um `rclone.conf` cifrado), o código só desiste aos 45 s. A mensagem é "Timeout: A unidade nao ficou pronta em 45 segundos." O motivo real se perde.

## O que muda

- O stderr do `rclone mount` é guardado: as últimas linhas em memória e, opcionalmente, um arquivo de log por cofre via `--log-file`.
- A espera termina assim que o processo sai, usando o sinal de `Wait` da demanda 001.
- A mensagem de erro traz as últimas linhas do stderr do rclone.

## O que fica de fora

- Política de log geral do programa.
- Traduzir mensagens do rclone.
- Tirar a senha do ambiente (demanda 004).

## Pronto quando

- [ ] Teste em `internal/core` com um executável falso que escreve `CRITICAL: teste` no stderr e sai com código 1: `MontarUnidade` devolve `false` em menos de 2 s e a mensagem contém `CRITICAL: teste`.
- [ ] No Windows, com `vfs_cache_mode` trocado para `invalido` em Configurações VFS: "Destrancar" mostra erro em menos de 5 s, com o texto do rclone.
- [ ] Nenhum caminho de `MontarUnidade` deixa processo vivo depois de devolver `false` (conferido no mesmo teste).

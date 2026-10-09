# 001 — O programa perde o processo do rclone logo depois de montar

- Estado: Aberta
- Risco: Alto — estabilidade
- Onde: `internal/core/montagem.go:processoAtivo`, `Status`, `MontarUnidade`
- Depende de: —

## Contexto

`processoAtivo` chama `p.Signal(os.Signal(nil))`. O Go recusa um sinal `nil` com `os: unsupported signal type`. Reproduzido com Go 1.24 no Linux. No Windows, `Process.Signal` só aceita `os.Kill`. Nos dois casos a função devolve falso para qualquer processo.

`Status()` apaga do mapa toda montagem com `processoAtivo` falso. Como `ObterMontagens`, `ObterLetraPorRemoto` e `ListarCofres` passam por `Status()`, o cofre sai do mapa na primeira atualização da tela, no máximo 3 s depois de montar.

Efeitos:

- O card volta a "Trancado" com a unidade ainda montada.
- O botão "Trancar" some. Clicar em "Destrancar" monta o mesmo cofre de novo em outra letra.
- `Encerrar` não desmonta nada. Os processos `rclone mount` ficam órfãos depois de sair.
- `MontarUnidade` também nunca percebe o processo morrendo durante a espera (ver 003).

## O que muda

- O `core` passa a saber se o processo vive pelo resultado de `cmd.Wait()`, chamado uma única vez em uma goroutine por montagem. Isso substitui sinal de teste.
- O mapa de montagens só perde um item quando essa goroutine informa que o processo terminou, ou quando a desmontagem termina.
- `Status()` deixa de apagar itens como efeito colateral de uma leitura.

## O que fica de fora

- Conferir se o ponto de montagem existe depois de montado (demanda 010).
- Mudar a desmontagem (demanda 002).
- Mostrar estados novos na tela (demanda 018).
- Recuperar montagens órfãs de execuções anteriores do programa.

## Pronto quando

- [ ] `rg -n "Signal\(os.Signal\(nil\)\)" internal/` não encontra nada.
- [ ] Um teste em `internal/core` inicia um processo de longa duração (por exemplo, `sleep 30` ou um binário de teste do próprio Go), registra como montagem e confere que `ObterMontagens()` ainda o lista depois de 5 chamadas seguidas.
- [ ] O mesmo teste mata o processo por fora e confere que, em até 2 s, `ObterMontagens()` não o lista mais.
- [ ] `go test -race ./internal/core/...` passa.
- [ ] No Windows com WinFsp: destrancar um cofre, esperar 10 s. O card continua "Destrancado" e o botão é "Trancar".
- [ ] No Windows: destrancar, clicar em "Sair" e conferir no Gerenciador de Tarefas que não sobra processo `rclone.exe`.

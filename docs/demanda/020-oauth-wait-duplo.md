# 020 — OAuth chama `cmd.Wait` duas vezes no mesmo processo

- Estado: Feita (0132f0b)
- Risco: Médio — estabilidade
- Onde: `internal/core/oauth.go:Iniciar` (goroutine de leitura), `Abortar`
- Depende de: 016

## Contexto

`Iniciar` sobe `rclone authorize` e abre uma goroutine que lê a saída e, no fim, chama `cmd.Wait()`. `Abortar` mata o processo e também chama `cmd.Wait()` no mesmo `exec.Cmd`. `exec.Cmd` não aceita `Wait` em duas goroutines: o segundo devolve erro ou disputa os mesmos campos (`ProcessState`, os pipes). O detector de corrida (`go test -race`) acusa.

Isso acontece toda vez que o usuário cancela a autorização, quando o laço de 120 s de `main.go` desiste e quando `Iniciar` é chamado de novo (ele começa com `Abortar`).

O teste `TestAbortarEncerraOProcesso` em `internal/core/oauth_test.go` (criado na demanda 016) está com `t.Skip` por causa disso.

## O que muda

- Um único dono para o `Wait` do processo de `authorize`. Por exemplo: só a goroutine de leitura chama `Wait` e fecha um canal de fim; `Abortar` mata o processo e espera esse canal. A forma exata fica para o PR, com o motivo escrito.
- `Abortar` só volta depois que o processo terminou.

## O que fica de fora

- Tempo limite do `authorize` (demanda 008).
- O laço de espera de 120 s em `main.go` e qualquer mudança na tela.
- O jeito de achar a URL e o token na saída do rclone.

## Pronto quando

- [ ] `TestAbortarEncerraOProcesso` sem `t.Skip`.
- [ ] `go test -race ./internal/core/...` passa, com esse teste rodando, em `ubuntu-latest` e `windows-latest` no CI.
- [ ] `rg -n "\.Wait\(\)" internal/core/oauth.go` mostra uma chamada só.
- [ ] Teste: `Iniciar` duas vezes seguidas deixa o primeiro `authorize` encerrado (pid conferido pelo rclone falso) e o segundo vivo até `Abortar`.

# 016 — Testes do core com executor de rclone substituível

- Estado: Feita (bc37337)
- Risco: Médio — estabilidade
- Onde: `internal/core/gerenciador.go`, `montagem.go`, `oauth.go` (todas as chamadas `exec.Command`)
- Depende de: —

## Contexto

Há dois testes: `TestCacheSenhas` e `TestConfigVfs`. Nada testa montagem, cofres, OAuth ou o contrato com o rclone. As chamadas usam `exec.Command(g.Executavel, ...)` direto, o que obriga cada teste a ter rclone real.

As demandas 001 a 015 pedem testes. Sem um ponto de troca do executável, cada uma vai inventar o seu.

## O que muda

- O caminho do executável e o diretório do app passam a ser injetáveis no `core`, para os testes apontarem para um executável falso compilado pelo próprio teste.
- Testes de caracterização registram o comportamento atual de `CriarCrypt`, `MontarUnidade`, `DesmontarUnidade` e da leitura do OAuth antes das correções. Onde o comportamento atual é defeito, o teste fica marcado com `t.Skip` e o número da demanda.
- O CI roda `go vet ./...` e `go test -race ./internal/core/...`.

## O que fica de fora

- Testes da GUI.
- Mudar comportamento. Esta demanda só cria a base de testes.

## Pronto quando

- [ ] `go test ./internal/core/...` roda sem rclone instalado.
- [ ] Há pelo menos um teste para cada função pública de `gerenciador.go`, `montagem.go` e `oauth.go`.
- [ ] Um workflow do GitHub Actions roda os comandos acima em `ubuntu-latest` e `windows-latest` a cada PR.

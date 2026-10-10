# 029 — Criar cofre falha quando a senha ofuscada começa com hífen

- Estado: Aberta
- Risco: Alto · uso
- Onde: `internal/core/criar_remoto.go:criarRemoto`
- Depende de: —

## Contexto

`criarRemoto` monta `rclone config create <nome> <tipo> <chave> <valor> …` com cada valor como um argumento solto. O crypt recebe `password` e `password2` já ofuscados por `rclone obscure`. O valor ofuscado é base64 de URL (`A–Z a–z 0–9 - _`) de um vetor aleatório, então o primeiro caractere é `-` em cerca de 1 de cada 64 criações (o dobro com uma segunda senha diferente).

Quando isso acontece, o rclone lê o valor como uma opção e recusa o comando. Reproduzido no box com rclone v1.60.1 e v1.75.2:

```
$ rclone config create t1 crypt remote /tmp/x password -3RPpB-dM90yplYzO40JSrYR1a-xgwdS
Fatal error: unknown shorthand flag: '3' in -3RPpB-dM90yplYzO40JSrYR1a-xgwdS
```

Para o usuário, criar ou conectar um cofre falha de vez em quando, sem motivo aparente, e tentar de novo com a mesma senha funciona.

Os argumentos também saem numa ordem diferente a cada chamada, porque vêm de um `map`. Isso não quebra nada, mas impede um teste de conferir a linha de comando exata.

## O que muda

- `criarRemoto` passa `--` logo depois de `config create`: `rclone config create -- <nome> <tipo> <chave> <valor> …`. Depois de `--` o rclone (cobra/pflag) não lê mais nada como opção. Conferido no box com v1.60.1 e v1.75.2: o valor `-3RP…` foi gravado como está no `rclone.conf`, e o rclone não ofuscou de novo um valor que já vinha ofuscado.
- Os pares chave/valor saem em ordem alfabética da chave.
- O rclone falso dos testes passa a recusar, como o real, um valor que começa com `-` antes de `--`.

## O que fica de fora

- Tirar a senha ofuscada e o token dos argumentos (demanda 005).
- Onde a senha fica guardada (demanda 004, ADR-0006).
- Outros comandos do rclone. Os que recebem valor do usuário (`lsjson`, `config delete`) começam pelo nome do cofre, e `ValidarNomeCofre` já recusa nome que começa com hífen.

## Pronto quando

- [ ] Teste: com `obscure` devolvendo um valor que começa com `-`, a chamada de `config create` é exatamente `config create -- cofre crypt directory_name_encryption true filename_encryption standard password -x… password2 -x… remote gdrive:pasta`, e o cofre é criado.
- [ ] Teste: o rclone falso recusa `config create cofre crypt password -x` (sem `--`), como o real.
- [ ] Teste: 500 senhas geradas no formato do `rclone obscure` (incluindo as que começam com `-`) criam o remoto, e o `rclone.conf` falso guarda o valor exato.
- [ ] Com o rclone real (v1.60.1 ou mais novo): `rclone config create -- t crypt remote /tmp/x password -3RPpB-dM90yplYzO40JSrYR1a-xgwdS` sai com 0 e grava `password = -3RPpB-dM90yplYzO40JSrYR1a-xgwdS`.

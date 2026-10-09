# 017 — Tirar a orquestração de `main.go` e as regras de `internal/gui`

- Estado: Aberta
- Risco: Médio — estabilidade e migração
- Onde: `main.go:acaoNovoCofre`, `acaoImportarCofre`, `destravarCofre`, `travarCofre`, `autoMontarCofres`; `internal/gui/wizards.go:DialogoNovoCofre`, `DialogoImportarCofre`; `internal/gui/seletor_pasta.go:splitCaminho`, `joinCaminho`
- Depende de: 016

## Contexto

O `core` não importa Fyne, mas os casos de uso não moram nele:

- `main.go` decide a ordem dos passos (OAuth, remoto base, crypt, gravação), o nome `<nome>_base`, o laço de 120 s do token (duplicado), o uso do cache de senha ao destrancar e ao trancar, e o auto-montar.
- `internal/gui/wizards.go` aplica senha ≥ 8, senha igual à confirmação, nome não vazio e `password2` padrão.
- `internal/gui/seletor_pasta.go` monta caminhos remotos.

Uma troca de interface (ADR-0007) teria de reescrever tudo isso no frontend.

## O que muda

- O `core` ganha uma operação por caso de uso, por exemplo `CriarCofre`, `ConectarCofre`, `Destrancar`, `Trancar` e `AutoMontar`. Cada uma recebe dados e devolve resultado ou erro, sem diálogo.
- Passos que precisam do usuário (abrir navegador, escolher pasta) entram como interface ou callback que a GUI implementa.
- `main.go` e `internal/gui` passam a só coletar entrada, chamar o `core` e mostrar o resultado.
- Comportamento igual ao de hoje, com os defeitos já corrigidos pelas demandas anteriores.

## O que fica de fora

- Trocar Fyne (019).
- Estados novos na tela (018).

## Pronto quando

- [ ] `main.go` tem menos de 120 linhas e não chama `exec`, `CriarRemoto`, `CriarCrypt` nem `Cofres.Adicionar`.
- [ ] `rg -n "len\(senha\) < 8|senha2 = senha" internal/gui main.go` não encontra nada.
- [ ] `go list -deps ./internal/core | rg fyne` não encontra nada.
- [ ] Testes de `CriarCofre` e `ConectarCofre` rodam com executável falso, sem GUI.

# 022 — A senha da sessão só sai quando o cofre trancou

- Estado: Feita (b70d5e9)
- Risco: Médio · segredos, uso
- Onde: `main.go:travarCofre`; `internal/core/montagem.go:DesmontarUnidade`; `internal/gui/janela_principal.go:criarCardCofre`
- Depende de: 002

## Contexto

`travarCofre` apaga a senha da sessão mesmo quando a desmontagem falha. A unidade continua montada, mas o programa já esqueceu a senha. O card não diz que o trancar falhou.

## O que muda

- Se a desmontagem não terminou, o card continua `Destrancado • X:\` e mostra `Não trancou: {motivo}`.
- A senha da sessão só é apagada quando o card chega a `Trancado`.

## O que fica de fora

- Os quatro estados do cofre (018). Esta demanda só trata do caso em que o trancar falha.
- Forçar a desmontagem de outro jeito.

## Pronto quando

- [ ] Teste: desmontagem que falha (rclone falso que não termina) deixa a senha da sessão no lugar, e `travarCofre` devolve erro.
- [ ] Teste: desmontagem que termina apaga a senha da sessão.
- [ ] Na tela (Windows): com um arquivo aberto na unidade, `Trancar` que falha deixa o card em `Destrancado • X:\` com `Não trancou: {motivo}`. Um novo `Destrancar` não pede a senha de novo.

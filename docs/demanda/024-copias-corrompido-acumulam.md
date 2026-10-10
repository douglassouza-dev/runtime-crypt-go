# 024 — Cópias `.corrompido` se acumulam

- Estado: Aberta
- Risco: Baixo · uso
- Onde: `internal/core/cofres.go:carregar`
- Depende de: 007

## Contexto

Com o `vaults.json` ilegível, cada abertura do programa guarda mais uma cópia `.corrompido-<data>`. Enquanto o arquivo não for consertado, as cópias se acumulam.

## O que muda

- Se já existe uma cópia `.corrompido` com o mesmo conteúdo, nenhuma nova é criada.

## O que fica de fora

- Apagar cópias antigas.
- Consertar o JSON automaticamente.

## Pronto quando

- [ ] Teste: abrir o gerenciador três vezes com o mesmo `vaults.json` ilegível deixa uma cópia `.corrompido` só.
- [ ] Teste: se o conteúdo ilegível muda, nasce uma cópia nova.

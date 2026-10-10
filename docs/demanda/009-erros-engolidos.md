# 009 — Erros engolidos viram lista vazia ou silêncio

- Estado: Aberta
- Risco: Médio — estabilidade
- Onde: `internal/core/gerenciador.go:ListarDiretoriosRemoto`, `ListarRemotos`, `ListarTodosRemotos`, `ListarRemotosDetalhado`, `ObterConfigRemoto`; `main.go:autoMontarCofres`, `main` (ramo `AcaoAutoIniciar`); `internal/gui/seletor_pasta.go:DialogoSeletorPastaRemota`
- Depende de: 008

## Contexto

- `ListarDiretoriosRemoto` devolve `nil` em erro, em tempo esgotado e em pasta vazia. O seletor mostra "Nenhuma subpasta encontrada. Você pode selecionar esta pasta." nos três casos. Com o token inválido, o usuário pode escolher a raiz achando que está vazia.
- As quatro funções de listagem de remotos devolvem `nil` em qualquer erro.
- `autoMontarCofres` ignora o resultado de `MontarUnidade`.
- `AdicionarAutoIniciar` e `RemoverAutoIniciar` devolvem `error`, e `main.go` descarta. No Windows, `RemoverAutoIniciar` devolve `nil` até quando falha.
- `AbrirExplorador` e `AbrirNavegador` devolvem erro que ninguém lê.

## O que muda

- As funções de listagem devolvem `([]T, error)`.
- O seletor distingue "vazia" de "erro". Pasta vazia mostra `Nenhuma subpasta aqui.` (021). Erro de listagem mostra `Não deu para listar as pastas: {motivo}`, com o botão `Tentar de novo` ao lado. Erro nunca aparece como `Nenhuma subpasta aqui.`.
- Os erros de auto-iniciar, auto-montar e de abrir Explorer ou navegador chegam ao usuário.

## O que fica de fora

- Log em arquivo.
- Tempo limite (008).

## Pronto quando

- [ ] `rg -n "return nil$" internal/core/gerenciador.go` não encontra retorno de erro disfarçado de lista vazia.
- [ ] Teste: `ListarDiretoriosRemoto` com executável falso que sai com código 1 devolve erro diferente de `nil`.
- [ ] Na tela: conectar existente com um remoto base de token inválido mostra `Não deu para listar as pastas: {motivo}` com `Tentar de novo`, nunca `Nenhuma subpasta aqui.`.
- [ ] `Tentar de novo` lista de novo a mesma pasta. Se der certo, a lista aparece no lugar da linha de erro.

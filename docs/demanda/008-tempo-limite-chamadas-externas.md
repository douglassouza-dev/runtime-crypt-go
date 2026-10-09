# 008 — Chamadas ao rclone sem tempo limite

- Estado: Aberta
- Risco: Médio — estabilidade
- Onde: `internal/core/gerenciador.go:testarRclone`, `ObscurecerSenha`, `CriarRemoto`, `RemoverRemoto`, `ListarRemotos`, `ListarTodosRemotos`, `ListarRemotosDetalhado`, `ObterConfigRemoto`; `internal/core/oauth.go:Iniciar`
- Depende de: —

## Contexto

Só `lsd` (30 s) e `mount` (45 s) têm limite. As outras chamadas usam `exec.Command` sem `context`. Se o rclone travar ou pedir algo no stdin (por exemplo, `rclone.conf` cifrado sem `RCLONE_CONFIG_PASS`), a goroutine fica parada para sempre. `testarRclone` roda até três vezes em `NovoGerenciador`, antes de a janela abrir.

`ListarDiretoriosRemoto` mata o processo no tempo esgotado sem chamar `Wait`.

`rclone authorize` não tem limite próprio. Só o laço de 120 s em `main.go` o encerra.

## O que muda

- Toda chamada ao rclone usa `exec.CommandContext` com limite. Os valores ficam em constantes nomeadas no `core`, com o motivo de cada um em comentário.
- Processo morto por tempo esgotado sempre passa por `Wait`.
- O tempo esgotado vira erro com a mensagem "tempo esgotado" e o comando, sem segredos.

## O que fica de fora

- Repetir chamadas (retry).
- Tornar os limites configuráveis pela tela.
- O limite da montagem, que já existe e é tratado na 003.

## Pronto quando

- [ ] `rg -n "exec\.Command\(" internal/core` só encontra `AbrirNavegador` e `AbrirExplorador`, que usam `Start` sem esperar.
- [ ] Teste com executável falso que dorme 10 min: cada função listada em "Onde" devolve erro de tempo esgotado dentro do limite + 1 s.
- [ ] O mesmo teste confere que não sobra processo filho.

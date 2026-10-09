# 005 — Token e senha ofuscada aparecem na lista de processos

- Estado: Aberta
- Risco: Médio — segredos
- Onde: `internal/core/gerenciador.go:CriarRemoto`, `CriarCrypt`; `main.go:acaoNovoCofre`, `acaoImportarCofre`
- Depende de: 004

## Contexto

`CriarRemoto` monta `rclone config create <nome> <tipo> <chave> <valor> ...`. O token OAuth (JSON com `access_token` e `refresh_token`) e as senhas ofuscadas vão nos argumentos. Qualquer processo do mesmo usuário vê os argumentos durante a chamada: Gerenciador de Tarefas, `ps`, `/proc/<pid>/cmdline`.

## O que muda

- Segredos deixam de ir em argumentos. Opções a avaliar na implementação: variáveis de ambiente por remoto (`RCLONE_CONFIG_<REMOTO>_<CHAVE>`), escrita direta do `rclone.conf` pelo programa ou stdin. A escolha e o motivo ficam no PR.

## O que fica de fora

- Onde os segredos ficam guardados (demanda 004).
- O ambiente do `rclone mount`, que também carrega segredo (tratado na 004).

## Pronto quando

- [ ] Teste em `internal/core` com um executável falso que grava os próprios argumentos em arquivo: depois de criar um cofre, o arquivo não contém o token nem a senha ofuscada.
- [ ] `rg -n '"token"' main.go internal/` só encontra o token fora de argumentos de `exec.Command`.

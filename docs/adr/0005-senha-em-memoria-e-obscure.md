# ADR-0005 — Senha da sessão em memória e senha do crypt ofuscada no `rclone.conf`

- Status: Aceita (registrada a posteriori). O [ADR-0006](0006-onde-ficam-os-segredos.md), Proposta, a substitui
- Data do registro: 2026-10-09
- Evidência: `internal/core/senhas.go`; `internal/core/gerenciador.go:CriarCrypt`, `ObscurecerSenha`; `internal/core/montagem.go:MontarUnidade`

## Contexto

O rclone crypt precisa de `password` e `password2` na configuração do remoto. O programa quer que o usuário digite a senha ao destrancar.

## Decisão (como está no código)

- Ao criar ou conectar, as senhas são ofuscadas com `rclone obscure` e gravadas no `rclone.conf`.
- Ao destrancar, a senha digitada fica em `CacheSenhas` (memória) e vai para o processo `rclone mount` em `RCLONE_CONFIG_PASS`.
- Ao trancar, ao falhar a montagem ou ao sair, a senha sai do cache.

## Consequências

- `rclone obscure` não é criptografia. Quem lê o `rclone.conf` recupera a senha com `rclone reveal`. Reproduzido.
- `RCLONE_CONFIG_PASS` só tem efeito com o `rclone.conf` cifrado, e o programa não cifra o arquivo. A senha digitada não é conferida. Reproduzido.
- Strings em Go não são zeradas. "Limpar" o cache só remove a referência do mapa.
- Essas consequências contradizem o README. A correção está na demanda 004.

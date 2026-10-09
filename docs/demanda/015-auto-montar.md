# 015 — Auto-montar nunca monta

- Estado: Aberta
- Risco: Baixo — uso
- Onde: `main.go:autoMontarCofres`; `internal/core/cofres.go:Atualizar`
- Depende de: 004, 009

## Contexto

`autoMontarCofres` exige `cofre.TemSenha`. O cache de senhas está sempre vazio quando o programa inicia. Nenhuma tela chama `Cofres.Atualizar` com `auto_montar`. O recurso anunciado no README não acontece.

## O que muda

Depende de onde a senha vai ficar (ADR-0006):

- Se houver cofre de senhas do SO, auto-montar lê a senha de lá, com consentimento explícito por cofre.
- Se não houver, auto-montar pede a senha ao iniciar, um cofre por vez.

Nos dois casos, a janela de cofres ganha uma opção por cofre para ligar e desligar o auto-montar.

## O que fica de fora

- Montar antes do login do usuário.
- Gravar a senha em `vaults.json`. Isso não é aceitável em nenhuma opção.

## Pronto quando

- [ ] Ligar auto-montar em um cofre, fechar e abrir o programa. O cofre monta, ou a senha é pedida, conforme o ADR-0006.
- [ ] Desligar e reabrir. O cofre não monta e nada é pedido.
- [ ] `vaults.json` não contém senha (conferido com `rg -n senha vaults.json`).

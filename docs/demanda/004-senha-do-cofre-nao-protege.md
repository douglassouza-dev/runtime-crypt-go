# 004 — A senha de destrancar não protege o cofre

- Estado: Aberta
- Risco: Alto — segredos
- Onde: `internal/core/gerenciador.go:CriarCrypt`; `internal/core/montagem.go:MontarUnidade`; `main.go:destravarCofre`
- Depende de: ADR-0006 aceito
- ADR: [0006 — Onde ficam os segredos dos cofres](../adr/0006-onde-ficam-os-segredos.md) (Proposta)
- Parte já separada: [030 — Senha errada não destranca](030-senha-errada-nao-destranca.md). A 030 confere a senha digitada contra a que está no `rclone.conf`, sem mudar onde ela fica. Esta demanda continua aberta para o resto.

## Contexto

`CriarCrypt` grava `password` e `password2` no `rclone.conf`, ofuscadas por `rclone obscure`. Ofuscar é reversível: `rclone reveal` devolve a senha. Reproduzido com rclone v1.60.1.

Ao destrancar, `MontarUnidade` passa a senha digitada em `RCLONE_CONFIG_PASS`. Essa variável só abre um `rclone.conf` cifrado, e o programa nunca cifra o arquivo. Com o arquivo em claro, o rclone ignora a variável. Reproduzido: `rclone cat` com `RCLONE_CONFIG_PASS=errada` leu o conteúdo do cofre.

Resultado:

- Qualquer senha não vazia destranca.
- Quem copia o `rclone.conf` tem a senha do cofre e o token da conta.
- O README diz o contrário ("Senhas ficam apenas em RAM — nunca salvas em disco").

## O que muda

Implementar a opção escolhida no ADR-0006. Qualquer opção tem de cumprir:

- a senha digitada ao destrancar é a que decide se o cofre abre;
- o `rclone.conf` não contém a senha do cofre de forma reversível sem um segredo que só o usuário tem.

## O que fica de fora

- Segredos em argumentos de linha de comando (demanda 005).
- Migrar cofres já criados. Se a opção escolhida exigir migração, ela vira outra demanda.
- O token OAuth, a menos que o ADR o inclua.
- Corrigir o README. Isso vem no mesmo PR, mas a decisão de texto não é escopo desta demanda.

## Pronto quando

- [ ] O ADR-0006 está com status Aceita.
- [ ] Com um cofre criado pelo programa, `rclone reveal` aplicado a qualquer valor do `rclone.conf` não devolve a senha do cofre.
- [ ] No Windows: destrancar com uma senha errada falha com mensagem de senha incorreta e não monta letra nenhuma.
- [ ] Destrancar com a senha certa monta e lista os arquivos.
- [ ] Teste em `internal/core` cobre senha certa e senha errada contra um remoto crypt local criado em pasta temporária.

# 025 — Trancar logo depois de gravar não espera o envio

- Estado: Feita (abdcd81)
- Risco: Alto · dados
- Onde: `internal/core/montagem.go:DesmontarUnidade`, `MontarUnidade`; `internal/core/trancar.go:Trancar`; `internal/gui/janela_principal.go:criarCardCofre`
- Depende de: 013, 018, 022

## Contexto

O `rclone mount` roda com `--vfs-cache-mode full` e `--vfs-write-back` de 5 s: um arquivo gravado só sobe para a nuvem uns segundos depois. `DesmontarUnidade` pede o fim do processo (Interrupt, depois Kill) sem conferir se ainda há envios pendentes. O rclone sai antes de enviar; o arquivo fica só no cache local e sobe apenas na próxima vez que o cofre for destrancado nesta máquina.

Reproduzido no Linux (FUSE3, rclone 1.60): gravar um arquivo e trancar 1 s depois deixa a pasta da nuvem vazia; esperar 7 s antes de trancar faz o arquivo cifrado aparecer.

## O que muda

- Trancar espera o rclone terminar os envios da VFS antes de encerrar o processo.
- Enquanto espera, a tela mostra `Enviando {n} arquivos…` (card e bandeja), e depois `Trancado`. Com um arquivo só, `Enviando 1 arquivo…`. Toda frase com contagem de arquivos segue a mesma regra de singular (`core.Arquivos`), inclusive os motivos `o envio de 1 arquivo falhou` e `o envio de {n} arquivos parou por {tempo}`.
- Se o envio falha, o cofre não é trancado: o card continua `Destrancado • …` com `Não trancou: {motivo}` (como na 022), e a senha da sessão fica.

## O que fica de fora

- Mudar `--vfs-write-back` ou o modo de cache.
- Enviar o que sobrou no cache de uma sessão anterior que terminou sem trancar (queda, desligamento).
- Barra de progresso em bytes.

## Pronto quando

- [ ] Teste: trancar com envios pendentes espera até a fila zerar e só então encerra o rclone; nenhum arquivo é perdido.
- [ ] Teste: envio que falha deixa o cofre montado, `Trancar` devolve erro com o motivo e a senha da sessão fica.
- [ ] Teste: enquanto espera, o estado do cofre traz quantos arquivos faltam, e a tela mostra `Enviando {n} arquivos…` (e `Enviando 1 arquivo…` com um só).
- [ ] Linux com FUSE3: gravar um arquivo e trancar 1 s depois; o arquivo cifrado está na nuvem quando `Trancar` volta.
- [ ] Na tela (Windows): gravar um arquivo grande e trancar; o card mostra `Enviando 1 arquivo…` e depois `Trancado`.

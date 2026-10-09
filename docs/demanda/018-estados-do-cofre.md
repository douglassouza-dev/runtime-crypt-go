# 018 — Estados do cofre visíveis: desmontado, montando, montado, falhou

- Estado: Aberta
- Risco: Médio — uso
- Onde: `internal/core/cofres.go:CofreStatus`, `Listar`; `internal/core/montagem.go:Status`; `internal/gui/janela_principal.go:criarCardCofre`, `atualizarCofres`; `internal/tray/tray.go:aoIniciar`, `AtualizarTooltip`
- Depende de: 001, 003, 010, 017

## Contexto

Hoje `CofreStatus.Montado` é um `bool`. A tela mostra "● Trancado" ou "● Destrancado • X:\". Não há "montando": durante até 45 s nada muda, e "Destrancar" continua clicável. Não há "falhou": o erro aparece em diálogo e some. A bandeja não mostra estado nenhum.

Regra: **montado só é verdadeiro quando o processo `rclone mount` está vivo E a letra ou a pasta de montagem existe.**

## O que muda

- O `core` expõe um tipo de estado com quatro valores: `desmontado`, `montando`, `montado` e `falhou`, mais o motivo da falha e o ponto de montagem.
- O `core` é o único lugar que decide o estado. A GUI só lê.
- Janela de cofres: cada card mostra o estado e o botão certo para ele. `montando` desabilita o botão. `falhou` mostra o motivo e oferece "Tentar de novo".
- Wizards novo cofre e conectar existente: mostram em que passo estão (autorizando, criando remoto, gravando) e o erro do passo que falhou.
- Bandeja: o tooltip resume quantos cofres há em cada estado e é atualizado a cada mudança.

## O que fica de fora

- Notificação do SO.
- Itens por cofre no menu da bandeja. Se forem pedidos, viram outra demanda.

## Pronto quando

- [ ] Teste em `internal/core`: processo vivo + ponto existe → `montado`; processo vivo + ponto ausente → `falhou`; processo morto + ponto existe → `falhou`; durante a espera → `montando`; sem processo → `desmontado`.
- [ ] `rg -n "Montado\s+bool" internal/core` não encontra nada.
- [ ] Na janela (Windows): durante o destrancar, o card mostra "montando" e o botão fica desabilitado. Um segundo clique não inicia outro `rclone.exe` (conferido no Gerenciador de Tarefas).
- [ ] Matar o `rclone.exe` pelo Gerenciador de Tarefas: em até 5 s o card mostra "falhou" com o motivo.
- [ ] Wizard novo cofre com OAuth cancelado no navegador: o wizard mostra o passo "autorizando" e depois o erro.
- [ ] Tooltip da bandeja muda de texto quando um cofre passa de `desmontado` para `montado`.

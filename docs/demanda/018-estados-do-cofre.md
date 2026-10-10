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

### O que a tela escreve

Os quatro valores são nomes do `core`. A tela fala em cofre. Card, bandeja e Wails usam as mesmas quatro frases:

| Estado no `core` | Frase na tela |
|---|---|
| `desmontado` | `Trancado` |
| `montando` | `Destrancando…` |
| `montado` | `Destrancado • X:\` (a letra ou a pasta real) |
| `falhou`, a montagem nunca subiu | `Não destrancou: {motivo}` |
| `falhou`, a montagem chegou a `montado` e caiu depois | `Caiu: {motivo}` |

`Não destrancou: {motivo}` só vale quando a montagem nunca subiu. Um cofre que chegou a `montado` e depois caiu mostra `Caiu: {motivo}`, com a mesma frase no card e na bandeja.

Motivos de queda na tela:

| Motivo no `core` | Motivo na tela |
|---|---|
| o processo do rclone terminou | `o rclone parou` |
| a letra ou a pasta de montagem sumiu | `a unidade X:\ sumiu` (a letra ou a pasta real) |

Se os dois acontecem, a tela mostra só `o rclone parou`. As strings de motivo do `core` podem continuar internas; a tradução fica na interface.

Os nomes `desmontado`, `montando`, `montado` e `falhou` não aparecem na tela.
- Janela de cofres: cada card mostra o estado e o botão certo para ele. `montando` desabilita o botão. `falhou` mostra o motivo e oferece "Tentar de novo".
- Wizards novo cofre e conectar existente: mostram em que passo estão e o erro do passo que falhou. A palavra "remoto" não aparece na tela.

| Passo no `core` | Frase na tela | Erro na tela |
|---|---|---|
| `autorizando` | `Autorizando no navegador…` | `Não deu para autorizar no navegador: {motivo}` |
| `criando remoto` | `Configurando o {provedor}…` | `Não deu para configurar o {provedor}: {motivo}` |
| `gravando` | `Gravando o cofre…` | `Não deu para gravar o cofre: {motivo}` |

- Diálogo de sucesso dos wizards: sem `Remoto base:` e `Remoto:`. Mostra, com as palavras do wizard, `Nome do cofre: {nome}`, `Provedor: {provedor}` e `Pasta: {pasta}` (a pasta escrita como no seletor: `/Backup/cofre`; no cofre novo, que não escolhe pasta, `/`).
- Erros do `core` com "remoto" no texto ganham texto da tela: nome já usado no rclone → `O nome '{nome}' já está em uso no rclone. Escolha outro nome para o cofre.`; falha ao ler a configuração → `Não deu para conferir a configuração do rclone: {motivo}`. Ainda podem aparecer, sem tradução, mensagens que vêm direto da saída do rclone (em inglês, com "remote").
- No Windows, o motivo de queda "ponto de montagem não respondeu" aparece como `a unidade X:\ não respondeu`.
- Bandeja: o tooltip resume quantos cofres há em cada estado e é atualizado a cada mudança.

## O que fica de fora

- Notificação do SO.
- Itens por cofre no menu da bandeja. Se forem pedidos, viram outra demanda.

## Pronto quando

- [ ] Teste em `internal/core`: processo vivo + ponto existe → `montado`; processo vivo + ponto ausente → `falhou`; processo morto + ponto existe → `falhou`; durante a espera → `montando`; sem processo → `desmontado`.
- [ ] `rg -n "Montado\s+bool" internal/core` não encontra nada.
- [ ] Na janela (Windows): durante o destrancar, o card mostra `Destrancando…` e o botão fica desabilitado. Um segundo clique não inicia outro `rclone.exe` (conferido no Gerenciador de Tarefas).
- [ ] Matar o `rclone.exe` pelo Gerenciador de Tarefas com o cofre destrancado: em até 5 s o card e o tooltip da bandeja mostram `Caiu: o rclone parou`.
- [ ] Destrancar sem o WinFsp (ou com outra falha que impede a unidade de subir): o card mostra `Não destrancou: {motivo}` e o botão `Tentar de novo`.
- [ ] Teste: `falhou` de uma montagem que nunca subiu vira `Não destrancou: {motivo}`; de uma que caiu vira `Caiu: o rclone parou` ou `Caiu: a unidade X:\ sumiu`; com processo morto e unidade sumida, só `Caiu: o rclone parou`.
- [ ] Wizard novo cofre com OAuth cancelado no navegador: o wizard mostra `Autorizando no navegador…` e depois `Não deu para autorizar no navegador: {motivo}`.
- [ ] Tooltip da bandeja usa as mesmas frases do card e muda quando um cofre passa de `Trancado` para `Destrancado • X:\`.
- [ ] Teste `TestDialogoDeSucessoSemRemoto` e `TestErrosDoCoreSemRemoto` em `internal/gui`: diálogo de sucesso e erros do core sem "remoto".
- [ ] `rg -n "desmontado|montando|\"montado\"|falhou" internal/gui internal/tray` não encontra texto exibido ao usuário.

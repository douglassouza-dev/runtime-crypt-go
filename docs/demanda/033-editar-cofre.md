# 033 — Editar cofre: nome, app do Google, reconectar e remover

- Estado: Aberta
- Risco: Alto · dados, segredos
- Onde: `internal/gui/janela_principal.go` (botão `Editar` no card); tela nova em `internal/gui` (editar cofre); `internal/core/cofres.go:Cofre`, `GerenciadorCofres.Remover`, `Atualizar`; `internal/core/gerenciador.go:RemoverRemoto`; `internal/core/cache_vfs.go:pastaCacheRclone`, `pendentesNoCacheVfs`; `internal/core/montagem.go:pastaDoCofre`, `removerPastaVazia`; `internal/core/oauth.go:GerenciadorOAuth.Iniciar`; `internal/core/casos_de_uso.go:Destrancar`, `EstadoDoCofre`
- Depende de: 006, 026, 031, 032

## Contexto

O app não tem como mudar nada num cofre depois de criado. A 031 registra: "Remover e editar cofre não existem no app hoje." O que existe no código:

- `GerenciadorCofres.Remover(nome)` (`internal/core/cofres.go`) tira o cofre de `vaults.json`, mas nenhuma tela chama.
- `GerenciadorCofres.Atualizar` só muda `auto_montar` e `caminho_cripto`.
- `RemoverRemoto` (`internal/core/gerenciador.go`) roda `rclone config delete <nome>`. Hoje só `CriacaoCofre.Desfazer` (006) chama, para limpar uma criação que falhou.

Por isso, três coisas não têm saída hoje:

- **Token vencido.** Um cofre cujo token venceu, por exemplo um app do Google em modo Testing (7 dias, ver 032), não tem como ser autorizado de novo.
- **App compartilhado do rclone.** Um cofre do Google Drive que usa o app compartilhado não tem como trocar para o app próprio. A 032 só cobre cofres novos, e o rclone avisa que o app compartilhado para de funcionar em 2026 ([rclone, Making your own client_id](https://rclone.org/drive/#making-your-own-client-id)).
- **Cofre que o usuário não quer mais.** Ele continua na lista, com a senha e o token no `rclone.conf`.

### Onde o nome do cofre aparece hoje

O nome do cofre (`Cofre.Nome`) é também o nome do remoto crypt. Ele é a chave de quase tudo:

| Onde | Como | Arquivo |
|---|---|---|
| `vaults.json` | `nome` e `remoto_base` (`<nome>_base:` ou `<nome>_base:<pasta>`) | `cofres.go:Cofre` |
| `rclone.conf` | seção `[<nome>]` (crypt) e `[<nome>_base]` (provedor); o crypt aponta para `<nome>_base:` | `criar_remoto.go`, `criacao.go:NomeRemotoBase` |
| Montagem | `rclone mount <nome>:` e `--volname "RuntimeCrypto (<nome>)"` | `montagem.go` |
| Pasta de montagem (Linux, macOS) | `~/RuntimeCrypto/<nome>` | `montagem.go:pastaDoCofre` |
| Cache da VFS | `{cache}/vfs/<nome>/` e `{cache}/vfsMeta/<nome>/` (pasta do rclone, 026) | `cache_vfs.go` |
| Senha da sessão, estado, falhas | chave `<nome>` | `senhas.go`, `montagem.go:EstadosPorRemoto` |
| Conferir senha (030) | lê a seção `[<nome>]` do `rclone.conf` | `conferir_senha.go` |

O rclone não tem comando para renomear um remoto sem perguntas: a lista de `rclone config` tem `create`, `update`, `delete`, `reconnect`, `disconnect` e outros, mas não `rename` ([rclone config](https://rclone.org/commands/rclone_config/)). Renomear só existe no menu interativo.

## O que muda

### Quando dá para editar (aprovado pela UI)

- Cada card ganha o botão `Editar`.
- Ele só funciona com o cofre `Trancado`, isto é, `EstadoDesmontado` sem processo do rclone. Em qualquer outro estado (destrancado, destrancando, trancando com envio pendente, caiu, não destrancou), o botão aparece desabilitado com `Tranque o cofre para editar.` Isso resolve as dúvidas sobre cache e montagem: nada está montado enquanto a tela está aberta.
- No core, cada operação desta demanda confere de novo o estado antes de começar e recusa se o cofre não está desmontado. A tela pode estar velha.
- **Modo só leitura (031):** `Editar` e `Remover cofre` ficam desabilitados, como `Adicionar Cofre`. No core, renomear, trocar o app, reconectar e remover recusam antes de chamar o rclone, com a frase da faixa da 031. Nenhum desses caminhos toca no `rclone.conf` nesse modo.

### A tela (aprovado pela UI)

A tela começa pequena, com três partes:

1. **Nome.** Campo com o nome atual.
2. **App do Google.** Só para cofres do Google Drive. São os campos da 032: `Usar meu próprio app do Google`, `Client ID`, `Client secret` e o link do guia do rclone. Se o cofre já usa um app próprio, o `Client ID` aparece preenchido e o `Client secret` vazio, com o placeholder **(proposta)** `Deixe vazio para manter o atual`. O secret gravado nunca volta para a tela.
3. **`Reconectar`.** Autoriza o OAuth de novo. Aparece para Google Drive, OneDrive e Dropbox, não para S3 nem Pasta Local.

Embaixo, separado, `Remover cofre`, em vermelho.

Botões da tela (**proposta**): `Cancelar` e `Salvar`. `Salvar` só fica habilitado se o nome mudou ou se os campos do app do Google mudaram.

### Reconectar e trocar o app do Google

- `Reconectar` sem mudar o app: `rclone authorize <tipo>`, ou `rclone authorize drive <client_id> <client_secret>` se o `<nome>_base` já tem um par. Depois, `rclone config update -- <nome>_base token <JSON>`.
- Trocar para o app próprio, ou trocar de app: só acontece junto com `Reconectar`. `Salvar` com o par mudado roda o mesmo fluxo e grava `client_id`, `client_secret` e `token` juntos, num `config update` só. O rclone diz que pôr o par sem um token novo não basta: "This part is important otherwise rclone won't actually use your new client_id/client_secret." ([fórum do rclone, ACTION REQUIRED](https://forum.rclone.org/t/google-drive-and-google-photos-users-action-required/54005)).
- Voltar para o app compartilhado (apagar o par) também passa por `Reconectar`. O `config update` grava `client_id` e `client_secret` vazios e o token novo.
- Se a autorização falha, nada é gravado: o `<nome>_base` fica como estava, com o token antigo. Os erros e frases são os da 032 (`Não deu para autorizar: o Google recusou esse app.` para `invalid_client`), mais os da 027.
- O remoto crypt, a senha e `vaults.json` não mudam.
- Antes do código, conferir com o rclone real:
  - se `config update` mantém o resto da seção (`scope`, `team_drive`, `drive_id` do OneDrive);
  - se ele precisa de `--non-interactive` para não tentar abrir o navegador por conta própria ([rclone config create](https://rclone.org/commands/rclone_config_create/) descreve essa flag "for use by applications that wish to configure rclone themselves").
- Frase de sucesso (**proposta**): `Cofre reconectado.`

### Renomear

Decisão: renomear muda o nome que o usuário vê. Os nomes dos remotos no `rclone.conf` e as pastas do cache continuam os de antes.

- `vaults.json` ganha o campo `remoto`, o nome do remoto crypt. Um cofre gravado antes da 033, sem o campo, usa `remoto = nome` ao ler. Nada é regravado só por abrir o app (031).
- Tudo que fala com o rclone passa a usar `remoto`, e não `nome`: montar, conferir a senha (030), estado, falhas, senha da sessão, cache (026, 028) e o seletor. A tela usa `nome`.

| O quê | Muda? | Por quê |
|---|---|---|
| `vaults.json`: `nome` | Sim | É o que o usuário pediu |
| `vaults.json`: `remoto`, `remoto_base` | Não | Apontam para o `rclone.conf`, que não muda |
| `rclone.conf`: `[<antigo>]` e `[<antigo>_base]` | Não | Ver abaixo |
| `--volname` | Sim, `RuntimeCrypto (<novo>)` | É só o rótulo da unidade |
| Pasta de montagem (Linux, macOS) | Sim, `~/RuntimeCrypto/<novo>` | O usuário procura pelo nome que vê. A pasta antiga é apagada só se estiver vazia (`removerPastaVazia`); se não estiver, fica e vai para o log |
| Cache da VFS | Não | Fica em `vfs/<remoto>` e `vfsMeta/<remoto>`. Arquivos que não subiram continuam sendo retomados na próxima montagem (026) |

Por que não renomear os remotos:

- O rclone não tem `config rename`. Seria preciso ler as seções com `config dump`, recriar com `config create --no-obscure` (a senha já vem ofuscada, ver [rclone config create](https://rclone.org/commands/rclone_config_create/)), apontar o crypt novo para o base novo e apagar os antigos. São quatro passos que podem parar no meio. A senha ofuscada e o token passariam pelos argumentos do processo (005).
- O cache da VFS fica numa pasta com o nome do remoto. Trocar o remoto deixaria órfão o que ainda não subiu, ou exigiria mover pastas internas do rclone.

Riscos da decisão:

- O `rclone.conf` fica com o nome antigo. Quem usa o rclone na linha de comando precisa saber disso.
- Um cofre novo com o nome antigo é recusado, porque o remoto antigo ainda existe (006, `ErroNomeNoRclone`). Frase (**proposta**): `Esse nome ainda é usado pelo cofre {nome atual}. Escolha outro.`
- O nome novo passa pela mesma validação de `ValidarNomeCofre`. Ele não pode repetir o `nome` nem o `remoto` de outro cofre (sem diferenciar maiúsculas, como a 006). Ele pode repetir o próprio `remoto` do cofre, para voltar ao nome antigo.

A alternativa, renomear tudo, fica como pergunta 1.

### Remover cofre

#### Confirmação (aprovado pela UI)

- Remover pede confirmação com o nome do cofre: o usuário digita o nome, e o botão de confirmar só habilita quando o texto bate.
- Texto do diálogo (**proposta**):
  - título: `Remover {nome}?`
  - corpo: `O cofre sai da lista e do rclone. Os arquivos continuam no {provedor}, cifrados. Para abrir de novo, você vai precisar da senha do cofre.`
  - campo: `Digite {nome} para confirmar`
  - botões: `Cancelar` e `Remover`, em vermelho.

#### Bloqueio por arquivos que não subiram (aprovado pela UI)

- Antes de tudo, conta os arquivos pendentes no cache com `pendentesNoCacheVfs(pastaCacheRclone(args, env), remoto)`, a mesma conta da 026. Usa os mesmos argumentos de VFS da montagem, então vale o `cache_dir` do `vfs.json`.
- Com pendentes, não remove nada:
  - `Não removeu: {n} arquivos ainda não subiram.`
  - com 1: `Não removeu: 1 arquivo ainda não subiu.`
- Se não dá para ler o cache, também não remove (falha fechada, como a 026). Frase (**proposta**): `Não removeu: não deu para conferir os arquivos que faltam subir.`
- Para fazer subir o que falta, o usuário destranca o cofre: o rclone retoma o envio (026). A frase não diz isso; fica para a UI.

#### O que remover faz, nesta ordem

1. Confere de novo: cofre desmontado, sem modo só leitura (031) e sem pendentes no cache.
2. **Cache da VFS:** apaga `{cache}/vfs/<remoto>/` e `{cache}/vfsMeta/<remoto>/`. Sem pendentes, ali só há cópias de arquivos que já estão no provedor. Por que apagar:
   - no crypt montado, o cache guarda os arquivos já decifrados; deixá-los no disco depois de remover o cofre deixa o conteúdo legível sem senha;
   - ninguém mais monta esse remoto, então o rclone nunca limparia essa pasta.

   Só essas duas pastas, dentro da pasta de cache que `pastaCacheRclone` devolve. Se o caminho calculado não estiver dentro dela (nome com `..`, por exemplo), não apaga nada e para.
3. **Remoto crypt:** `rclone config delete <remoto>`, mas só se a seção é `type = crypt` e o `remote` dela começa com o nome do `remoto_base` do cofre. Se não bater (o usuário mexeu no `rclone.conf`), não apaga e para com erro.
4. **Remoto base:** `rclone config delete <remoto>_base`. Antes, `config dump` confere que nenhuma outra seção do `rclone.conf` usa `<remoto>_base:` como `remote`. Se outra usa (um alias do usuário, por exemplo), o base fica e isso vai para o log.
5. **`vaults.json`:** `GerenciadorCofres.Remover(nome)`. Fica por último: se algo antes falhou, o cofre continua na lista e dá para tentar de novo. Um remoto que já não existe no `rclone.conf` conta como removido.
6. **Pasta de montagem** (Linux, macOS): apaga `~/RuntimeCrypto/<nome>` só se estiver vazia (`removerPastaVazia`). No Windows não há pasta: a montagem é uma letra.
7. **Memória:** tira a senha da sessão (`Senhas.Limpar`) e as falhas guardadas desse remoto.

Por que apagar os remotos do `rclone.conf`, e não só tirar da lista:

- Quem lê o `rclone.conf` tem a senha do cofre e o token da conta (05-modelo-de-dados, "Onde ficam os segredos"). Deixar as seções lá depois de o usuário pedir para remover deixa esses segredos sem dono.
- Com as seções lá, criar ou importar de novo com o mesmo nome é recusado (006).
- Um app de cofres que tira o cofre da lista e deixa o acesso configurado surpreende quem removeu para se livrar dele.

#### O que remover NÃO faz

- **Nunca apaga nada no provedor.** Não roda `delete`, `purge`, `rmdir` nem nada que fale com a nuvem além do OAuth. Os arquivos cifrados continuam no Google Drive, OneDrive, Dropbox, S3 ou na pasta local, e dá para conectar de novo pelo `Importar Cofre Existente`, com a senha (e a senha 2, se for diferente).
- Não revoga o acesso do app na conta do provedor. O rclone tem `config disconnect`, mas isso fica fora (ver "O que fica de fora").
- Não apaga a pasta local da Pasta Local (os dados cifrados ficam nela).
- Não mexe em `vfs.json`: a configuração VFS é uma só, para todos os cofres.
- Não mexe no `Abrir ao ligar o computador` (atalho do sistema, 031).
- Não guarda cópia da senha em lugar nenhum. Se o usuário esqueceu a senha, os dados no provedor ficam ilegíveis para sempre. O texto de confirmação diz que a senha vai ser necessária.

## O que fica de fora

- Renomear os remotos no `rclone.conf` (ver pergunta 1).
- Mudar senha, provedor, pasta no provedor ou opções do crypt.
- `rclone config disconnect` para revogar o token ao remover.
- Editar `auto_montar` nesta tela (hoje ele não tem tela; a 015 cuida de auto-montar).
- Editar com o cofre destrancado.
- Desfazer uma remoção.

## Perguntas em aberto

1. **Renomear (Douglas):** a decisão aqui é mudar só o nome visto, sem tocar no `rclone.conf` nem no cache. Serve, ou o nome do remoto também deve mudar, com os riscos descritos acima?
2. **Cofre que não destrancou (UI):** um cofre em `Não destrancou: ...` não tem rclone rodando. `Editar` fica habilitado nesse estado? O caso mais comum é justamente o token vencido, que se resolve com `Reconectar`. Do jeito aprovado, só `Trancado` habilita.
3. **Arquivos que não subiram (UI):** a frase de bloqueio deve dizer como resolver (destrancar para o envio terminar)?
4. **Frases marcadas como proposta (UI):** placeholder do secret, botões, `Cofre reconectado.`, o diálogo de remover, o nome em uso e o cache ilegível.

## Pronto quando

- [ ] Teste: `Editar` só habilita com o cofre `Trancado`; nos outros estados, desabilitado com `Tranque o cofre para editar.`. No modo só leitura (031), `Editar` e `Remover cofre` desabilitados, e o core recusa sem chamar o rclone.
- [ ] Teste com o rclone falso: `Reconectar` sem par chama `authorize <tipo>` e `config update -- <nome>_base token ...`; com par, `authorize drive <id> <secret>` e um `config update` com os três. Autorização que falha não chama `config update`.
- [ ] Teste: o app do Google só aparece para cofres do Google Drive; o secret gravado nunca aparece na tela nem no log.
- [ ] Teste: renomear muda só `nome` em `vaults.json` (e grava `remoto`); o `rclone.conf` falso e as pastas do cache ficam iguais; destrancar depois do nome novo monta `<remoto>:` com `--volname "RuntimeCrypto (<novo>)"` e (Linux) em `~/RuntimeCrypto/<novo>`.
- [ ] Teste: `vaults.json` antigo, sem `remoto`, abre com `remoto = nome` e não é regravado.
- [ ] Teste: nome novo igual ao `nome` ou ao `remoto` de outro cofre é recusado; voltar ao próprio nome antigo é aceito.
- [ ] Teste: remover com 1 e com 3 pendentes no cache recusa com a frase aprovada (singular e plural), sem chamar o rclone nem mexer em `vaults.json`; cache ilegível também recusa.
- [ ] Teste: remover sem pendentes apaga `vfs/<remoto>` e `vfsMeta/<remoto>` (e nada fora delas), chama `config delete <remoto>` e `config delete <remoto>_base`, tira de `vaults.json` e apaga a pasta de montagem vazia.
- [ ] Teste: crypt que não aponta para o base do cofre não é apagado; base usado por outra seção fica; falha no meio deixa o cofre na lista, e a segunda tentativa termina.
- [ ] Teste: nenhum comando do rclone que fala com o provedor (`delete`, `purge`, `rmdir`, `deletefile`, `cleanup`) aparece em `chamadas.log` em nenhum caminho desta demanda.
- [ ] Teste da tela: o diálogo de remover só habilita o botão quando o texto digitado é igual ao nome.
- [ ] Na tela, no Windows: `Reconectar` num cofre do Google Drive; trocar para o app próprio; renomear e destrancar; remover e conferir que os arquivos continuam no Drive e que `Importar Cofre Existente` abre de novo com a senha.

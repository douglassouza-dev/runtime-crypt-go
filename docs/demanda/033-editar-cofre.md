# 033 — Editar cofre: nome, app do Google, reconectar e remover (deste computador ou também do provedor)

- Estado: Aberta
- Risco: Alto · dados, segredos
- Onde: `internal/gui/janela_principal.go` (botão `Editar` no card); tela nova em `internal/gui` (editar cofre); `internal/core/cofres.go:Cofre`, `GerenciadorCofres.Remover`, `Atualizar`; `internal/core/gerenciador.go:RemoverRemoto`; `internal/core/cache_vfs.go:pastaCacheRclone`, `pendentesNoCacheVfs`; `internal/core/montagem.go:pastaDoCofre`, `removerPastaVazia`; `internal/core/oauth.go:GerenciadorOAuth.Iniciar`; `internal/core/casos_de_uso.go:Destrancar`, `EstadoDoCofre`, `CriarCofre` (crypt na raiz da conta)
- Depende de: 006, 026, 028, 031, 032

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
- Ele funciona com o cofre `Trancado` (`EstadoDesmontado`) e com o cofre em `Não destrancou: ...` (`EstadoFalhou` sem `Caiu`): nos dois não há processo do rclone. Em `Não destrancou`, a tela abre com `Reconectar` em destaque, porque o caso mais comum é o token vencido (027, 032).
- Em qualquer outro estado (destrancado, destrancando, trancando com envio pendente, caiu), o botão aparece desabilitado com `Tranque o cofre para editar.` Isso resolve as dúvidas sobre cache e montagem: nada está montado enquanto a tela está aberta.
- No core, cada operação desta demanda confere de novo o estado antes de começar e recusa se há processo do rclone para o cofre. A tela pode estar velha.
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

#### As duas opções (aprovado pela UI)

`Remover cofre` abre um diálogo com duas opções, nesta ordem:

1. `Só deste computador`. **Vem marcada.** É a remoção local descrita abaixo: nada muda no provedor.
2. `Deste computador e do {provedor}`. Primeiro apaga os arquivos do cofre no provedor; só se isso terminar por completo, faz a remoção local.

O app nunca marca a segunda opção sozinho, nem lembra a última escolha.

Não existe a opção "só do provedor", que apagaria os arquivos e manteria o cofre no app. Ela deixaria na lista um cofre vazio, que destranca e mostra uma pasta sem nada, sem que o usuário saiba por quê. Quem quer recomeçar no mesmo provedor remove o cofre e cria outro.

#### Confirmação

- Nas duas opções, o usuário digita o nome do cofre, e o botão só habilita quando o texto bate (aprovado).
- Título (**proposta**): `Remover {nome}?`
- Com `Só deste computador` marcada:
  - logo abaixo da pergunta (aprovado): `Os arquivos no {provedor} continuam lá.`
  - para Pasta Local, no lugar dessa linha (**proposta**): `Os arquivos na pasta {caminho} continuam lá.`
  - corpo (**proposta**): `O cofre sai da lista e do rclone. Para abrir de novo, você vai precisar da senha do cofre.`
  - botão (**proposta**): `Remover`, em vermelho.
- Com `Deste computador e do {provedor}` marcada:
  - a linha `Os arquivos no {provedor} continuam lá.` some;
  - o resumo do que vai ser apagado aparece no lugar (ver abaixo);
  - botão (aprovado): `Excluir do {provedor}`, em vermelho, habilitado só depois do nome digitado.
- Campo (**proposta**): `Digite {nome} para confirmar`. Botão `Cancelar` nas duas opções.

#### Por que não pede a senha do cofre de novo (aprovado pela UI)

- Apagar no provedor não usa a senha do cofre: o rclone lista e apaga com o token do provedor. A senha não protege nada aqui.
- Hoje, quem usa este computador já alcança o remoto pelo `rclone.conf`, que guarda o token e a senha ofuscada (05-modelo-de-dados, ADR-0006). Pedir a senha seria uma barreira que não existe fora do app.
- Quando o ADR-0006 for decidido, essa regra é revista junto.

#### Bloqueio por arquivos que não subiram (aprovado pela UI)

- Vale para as duas opções. Antes de tudo, conta os arquivos pendentes no cache com `pendentesNoCacheVfs(pastaCacheRclone(args, env), remoto)`, a mesma conta da 026. Usa os mesmos argumentos de VFS da montagem, então vale o `cache_dir` do `vfs.json`.
- Com pendentes, não remove nem apaga nada:
  - `Não removeu: {n} arquivos ainda não subiram. Destranque o cofre para eles terminarem de subir.`
  - com 1: `Não removeu: 1 arquivo ainda não subiu. Destranque o cofre para ele terminar de subir.`
- Se não dá para ler o cache, também não remove (falha fechada, como a 026). Frase (**proposta**): `Não removeu: não deu para conferir os arquivos que faltam subir.`

#### Apagar no provedor: o comando

**Decisão: `rclone delete --rmdirs <remoto>:`, sempre pelo remoto crypt. Nunca `purge`, e nunca no remoto base.**

O motivo está em como os cofres são criados hoje. `CriarCofre` grava `remoto_base` como `<nome>_base:`, sem pasta (`casos_de_uso.go:CriarCofre`). O crypt fica na **raiz da conta**, misturado com os outros arquivos do usuário. A doc do crypt chama isso de "not recommended" ([rclone crypt, --crypt-remote](https://rclone.org/crypt/#crypt-remote)).

- **`purge` apagaria a conta inteira.** A doc diz que `purge` remove "the path and all of its contents" e "does not obey include/exclude filters - everything will be removed" ([rclone purge](https://rclone.org/commands/rclone_purge/)). No crypt, o `Purge` repassa para o remoto de baixo o caminho cifrado (`do(ctx, f.cipher.EncryptDirName(dir))`, em [backend/crypt/crypt.go](https://github.com/rclone/rclone/blob/master/backend/crypt/crypt.go)). Na raiz, esse caminho é vazio, então o rclone apagaria a raiz de `<nome>_base:`, isto é, todo o Drive do usuário. O mesmo vale para `purge` ou `delete` direto no base.
- **`delete` pelo crypt vê quase só o que é do cofre.** O crypt pula nomes que não consegue decifrar ("By default, rclone will just log a NOTICE and continue as normal", [--crypt-strict-names](https://rclone.org/crypt/#crypt-strict-names)), e os cofres do app usam `filename_encryption = standard` e `directory_name_encryption = true` (`constantes.go:ConfiguracoesCryptPadrao`). "Quase" porque alguns nomes que não são do cofre passam pela decifragem (ver "Cofre na raiz" e a condição 3 abaixo). Por isso a exclusão no provedor é bloqueada para cofres na raiz e quando outros arquivos dividem a pasta do cofre.
- `delete` "only deletes files but leaves the directory structure alone"; com `--rmdirs` ele "removes empty directories but leaves root intact" ([rclone delete](https://rclone.org/commands/rclone_delete/)). A raiz do crypt (a raiz da conta, ou a pasta do cofre importado) nunca é apagada por esse comando.
- Cofre importado (`remoto_base` = `<nome>_base:<pasta>`): depois do `delete`, roda `rclone rmdir <nome>_base:<pasta>`, que só apaga a pasta se ela estiver vazia. Isso nunca roda quando `<pasta>` é vazio (raiz).
- `--dry-run` não é usado: o resumo antes vem do `rclone size`.

Risco que fica: dois cofres com a mesma senha e a mesma senha 2 no mesmo lugar da mesma conta decifram os nomes um do outro. O `delete` de um apagaria os arquivos do outro. Por isso existe o bloqueio por arquivos em comum, mais abaixo.

#### Cofre na raiz, ou com outros arquivos na pasta (aprovado pela UI)

A exclusão no provedor é **sempre bloqueada** quando o caminho do cofre é a raiz da conta (o que vem depois de `:` no `remoto_base` é vazio), com ou sem outros cofres. Hoje isso inclui todo cofre criado pelo app (`CriarCofre`). A opção `Deste computador e do {provedor}` aparece desabilitada com:

`Não dá para excluir do {provedor}: este cofre está na raiz junto com outros arquivos. Remova só deste computador e apague pelo site do {provedor}.`

`Só deste computador` continua disponível. Os cofres novos deixam de cair na raiz com a demanda 035 (uma pasta própria por cofre).

**Por quê: os nomes do próprio usuário, conferido no código.** Na raiz, os arquivos do cofre ficam junto com os arquivos comuns do usuário, sem cifra. O que o crypt faz com um nome desses (`DecryptFileName` e `decryptFileName` em `backend/crypt/cipher.go`):

- **`standard` com `filename_encoding = base32`** (o padrão do app): cada parte do caminho passa por `caseInsensitiveBase32Encoding.DecodeString`, que põe em maiúsculas e decodifica com `base32.HexEncoding` (alfabeto `0-9` e `A-V`). Depois, `decryptSegment` exige que o resultado tenha um número de bytes múltiplo de 16 (`nameCipherBlockSize`, o bloco do AES), diferente de zero e até 2048. Só então vem a conferência do preenchimento PKCS#7.
  - Um nome como `Foto 2024.jpg` é **recusado logo na decodificação**: espaço e `.` não estão no alfabeto. O mesmo vale para qualquer nome com ponto (extensão), espaço, hífen, sublinhado, acento ou as letras `w`, `x`, `y`, `z`.
  - Para chegar ao preenchimento, o nome precisa ter **só `0-9` e `a-v`** (maiúsculas ou minúsculas), **sem extensão**, e **exatamente 26, 52, 77, 103…** caracteres (16, 32, 48… bytes).
  - Desses, cerca de 1 em 256 passa (o último byte vale `0x01`; `0x02 0x02` e maiores somam bem menos), vira um nome sem sentido e seria apagado.
  - Taxa real para arquivos comuns: perto de zero, porque quase todo nome tem extensão ou espaço. Não é zero para nomes gerados por máquina sem extensão, como IDs e hashes em hexadecimal de 26, 52… caracteres. Pastas passam pela mesma conta, com `directory_name_encryption = true`; se uma pasta passa, o crypt entra nela e cada nome lá dentro passa de novo pela mesma conta.
- **`standard` com `base64`:** alfabeto `A-Z`, `a-z`, `0-9`, `-` e `_`, sem ponto nem espaço; precisa de 22, 43, 64… caracteres. A mesma ordem de grandeza.
- **`standard` com `base32768`:** usa um conjunto próprio de caracteres Unicode; nomes comuns em letras latinas são recusados na decodificação.
- **`obfuscate`:** `deobfuscateSegment` aceita qualquer nome no formato `<número>.<resto>`, ou `!.<resto>`, sem conferir nada. `2024.relatorio.pdf`, `1.jpg` e `01.mp3` passam e seriam apagados. É muito pior que `standard`.
- **`off`:** todo arquivo que termina no sufixo (`.bin`, ou qualquer arquivo com `suffix = none`) passa e seria apagado.

Por isso a raiz bloqueia sempre, sem tentar estimar caso a caso.

**Pasta com outros arquivos (cofre importado).** Fora da raiz, o app confere antes de liberar, junto com o resumo e no mesmo tempo limite:

- `rclone lsf --max-depth 1 <nome>_base:<pasta>` (o que existe na pasta) e `rclone lsf --max-depth 1 <remoto>:` (o que o crypt decifra);
- se o primeiro tem mais entradas que o segundo, há nomes que o crypt não decifra, isto é, arquivos que não são do cofre. Bloqueia com a mesma frase aprovada;
- erro ou tempo esgotado em qualquer das duas: bloqueia (na dúvida, bloqueia).

São duas listagens de uma pasta só, sem entrar nas subpastas, então é barato. Limites:

- olha só o primeiro nível; arquivos comuns colocados dentro das pastas cifradas do cofre não são vistos;
- um nome comum que passe pela decifragem (a conta acima) conta como do cofre e não é percebido.

Os dois casos são raros e ficam nos riscos.

#### Bloqueio por arquivos em comum com outro cofre (aprovado pela UI)

Quando outro cofre do `vaults.json` pode usar os mesmos arquivos no provedor, a opção `Deste computador e do {provedor}` fica desabilitada, com:

`Não dá para excluir: o cofre {outro} usa os mesmos arquivos no {provedor}.`

`Só deste computador` continua disponível. O bloqueio é o padrão porque, na dúvida, apagar pode destruir os dados de outro cofre, e não apagar não destrói nada. Douglas ainda pode mudar isso (pergunta 2).

**Quando dois cofres "usam os mesmos arquivos".** Os dois têm de bater nas três condições (a terceira está mais abaixo, em "Condição 3"):

1. **Mesma conta, até onde dá para saber.** Lida da seção `[<remoto>_base]` de cada um no `rclone.conf` (`config dump`):
   - tipo do remoto base diferente (`drive` e `onedrive`, por exemplo): contas diferentes, não bloqueia;
   - Google Drive: se os dois têm `team_drive` preenchido e diferente, são drives diferentes. O mesmo vale para `root_folder_id` preenchido e diferente nos dois. Em qualquer outro caso (os dois vazios, ou só um preenchido), conta como a mesma conta;
   - OneDrive: `drive_id` é gravado na configuração ([rclone onedrive](https://rclone.org/onedrive/)). `drive_id` diferente: contas diferentes; igual ou ausente em algum dos dois: a mesma;
   - Dropbox: a configuração não guarda nada que identifique a conta. Conta como a mesma;
   - S3: `endpoint` e `region` diferentes: armazenamentos diferentes; iguais ou ausentes: o mesmo. A chave de acesso não conta, porque duas chaves podem ser da mesma conta;
   - Pasta Local: o próprio caminho é a "conta"; a condição 2 decide.
2. **Mesmo caminho, ou um dentro do outro.** O caminho é o que vem depois de `:` no `remoto_base` (vazio para os cofres na raiz). Antes de comparar: tira `/` das pontas, junta `/` repetidas e, na Pasta Local, resolve o caminho absoluto. Compara pasta por pasta, sem diferenciar maiúsculas (OneDrive e Dropbox não diferenciam; na dúvida, o app trata o Drive igual). Bloqueia se os caminhos são iguais ou se um é o começo do outro (`Cofres` e `Cofres/A`). `Cofres` e `Cofres2` não bloqueiam.

**Por que não dá para separar as contas com certeza.** Dois remotos do Google Drive com tokens diferentes podem ser a mesma conta ou não, e nada na configuração diz qual. O rclone tem `rclone config userinfo`, que "prints the details of the person logged in to the cloud storage system" ([rclone config userinfo](https://rclone.org/commands/rclone_config_userinfo/)). Mas os backends `drive` e `dropbox` não implementam essa função: não há `UserInfo` em `backend/drive/drive.go` nem em `backend/dropbox/dropbox.go`. O ID real da pasta raiz do Drive o rclone só descobre por chamada à API, sem comando documentado para mostrar. Decisão: **bloquear de forma conservadora**. O app não faz chamada extra ao provedor para tentar separar as contas; usa só o que está no `rclone.conf`, e na dúvida conta como a mesma conta.

**Consequência.** A raiz está "dentro" de qualquer caminho. Um cofre importado em `Cofres/A` fica bloqueado por um cofre do mesmo tipo de conta que esteja na raiz, mesmo em outra conta. (O próprio cofre na raiz já é sempre bloqueado, ver "Cofre na raiz".) A demanda 035, com uma pasta por cofre, diminui esse efeito.

**Condição 3: o crypt de um cofre enxerga os arquivos do outro (aprovado, com um achado abaixo).** Comparada só com o `rclone.conf`, sem chamada ao provedor, revelando os valores ofuscados em Go como a 030 faz (`revelarObscuro`). Nenhum valor revelado vai para o log, para a tela ou para mensagem de erro; só o resultado (bloqueia ou não) e o nome do outro cofre.

Parâmetros do crypt que decidem se um cofre decodifica os nomes do outro, conferidos na doc e no código do rclone ([rclone crypt](https://rclone.org/crypt/), `backend/crypt/crypt.go` e `cipher.go`). Valor ausente vale o padrão do rclone:

| Parâmetro | Padrão do rclone | Entra na comparação? | Por quê |
|---|---|---|---|
| `password` | (obrigatório) | Sim, revelado | É a chave dos nomes |
| `password2` | vazio | Sim, revelado | É o sal. Vazio usa um sal fixo do rclone (`defaultSalt` em `cipher.go`: "If salt is "" then it uses a built in salt"). Então vazio com vazio é igual. O app grava `password2 = password` quando a senha 2 fica em branco (`criarCrypt`), então os cofres do app nunca têm `password2` vazio |
| `filename_encryption` | `standard` | Sim | Decide como o nome é lido (ver modos abaixo) |
| `directory_name_encryption` | `true` | Sim | Com `false`, as pastas ficam com o nome aberto. A doc diz: "If filename_encryption is "off" then this option will do nothing" |
| `filename_encoding` | `base32` | Sim | Muda como o nome cifrado vira texto no modo `standard` (`base32`, `base64`, `base32768`) |
| `suffix` | `.bin` | Só no modo `off` | No código, o sufixo só é usado quando o modo é `off` (`EncryptFileName` e `DecryptFileName` em `cipher.go`). `none` vira sufixo vazio |

Os três modos de nome, e o que cada um faz com os arquivos de outro cofre:

- **`off`:** a doc diz que ele "doesn't hide file names or directory structure" e "Adds a ".bin", or "suffix" extension only". No código, `DecryptFileName` no modo `off` só confere e tira o sufixo, sem senha. Um crypt com **outra senha** lista os `.bin` do outro cofre, e o `delete` apaga. **Se qualquer um dos dois cofres usa `off`, a senha não protege: bloqueia sempre que os caminhos se cruzam** (com o `suffix` igual ou não, porque com `none` todo arquivo serve).
- **`obfuscate`:** a doc diz que "Obfuscation cannot be relied upon for strong protection". No código, `deobfuscateSegment` aceita qualquer nome no formato `<número>.<texto>` com qualquer chave: não há conferência que falhe com a senha errada. Um crypt com outra senha lista os arquivos do outro com nomes trocados e apaga. **Tratado igual ao `off`: bloqueia sempre que os caminhos se cruzam.**
- **`standard`** (o padrão do app, `ConfiguracoesCryptPadrao`): com a senha errada, o nome decifrado só é aceito se o último byte for um preenchimento PKCS#7 válido (`pkcs7.Unpad` em `backend/crypt/pkcs7/pkcs7.go`). Não há outra conferência de integridade no nome.

**Achado: senhas diferentes não bastam no modo `standard`.** Com uma chave errada, o último byte decifrado é praticamente aleatório. Quando ele vale `0x01` (1 chance em 256), o preenchimento passa e o nome "decifra" para lixo. Os casos `0x02 0x02` e maiores somam bem menos. Ou seja: cerca de 1 em cada 256 nomes do outro cofre aparece para este crypt com um nome sem sentido. O `delete --rmdirs` apaga esses arquivos. Se for uma pasta, o crypt ainda entra nela, e cada nome lá dentro passa pela mesma loteria. Com pastas de nome aberto (`directory_name_encryption = false`), o crypt enxerga as pastas do outro cofre; o `rmdirs` tenta apagá-las, e o provedor recusa enquanto houver arquivo dentro.

Por isso, a regra fica assim:

1. `rclone.conf` sem a seção do crypt de algum dos dois, valor que não se revela, ou `filename_encryption` desconhecido: **bloqueia** (na dúvida, bloqueia).
2. Algum dos dois em `off` ou `obfuscate`: **bloqueia**.
3. Os dois em `standard`, com `password`, `password2`, `directory_name_encryption` e `filename_encoding` iguais: **bloqueia** (um enxerga tudo do outro).
4. Os dois em `standard`, com algum desses parâmetros diferente: **bloqueia. É o padrão recomendado pela UI e pelo time**, pelo achado acima; falta Douglas confirmar (pergunta 2). Liberar significa aceitar que cerca de 1 em 256 arquivos do outro cofre seja apagado. Não existe forma barata de evitar: conferir o conteúdo de cada arquivo exigiria baixar um bloco de cada um.

Na prática, com o item 4 bloqueando, a condição 3 não libera nenhum caso: ela documenta por que senhas diferentes não protegem. Se Douglas decidir liberar o item 4, os testes já cobrem a comparação.

**Onde roda.** Ao abrir o diálogo de remover, e de novo no core, logo antes do `delete`. O `vaults.json` pode ter mudado no meio.


#### Resumo antes de apagar

- Ao marcar a segunda opção, o app roda `rclone size --json <remoto>:` ([rclone size](https://rclone.org/commands/rclone_size/)) e mostra quantos arquivos e quantos bytes vão ser apagados.
- O custo não é baixo para cofres grandes: o `size` lista o cofre inteiro, pasta por pasta, e no Google Drive cada listagem gasta cota da API (032). Por isso:
  - roda só quando a segunda opção é marcada, com um tempo limite (**proposta**: 2 min);
  - enquanto roda, o botão `Excluir do {provedor}` fica desabilitado;
  - se falha ou passa do tempo, o resumo aparece sem os números, e o botão habilita mesmo assim. O usuário já digitou o nome, e o número é ajuda, não trava.
- Texto (**proposta**):
  - `Vai apagar {n} arquivos ({tamanho}) em {provedor}: {caminho}.`
  - sem números: `Vai apagar os arquivos deste cofre em {provedor}: {caminho}.`
  - `{caminho}` é a pasta do cofre no provedor. Cofre na raiz nunca chega aqui (é bloqueado antes).
  - seguido da linha de lixeira, conforme o provedor (abaixo).

#### Lixeira, por provedor

O app não passa nenhuma flag de lixeira: vale o que o provedor e o `rclone.conf` disserem. Antes de mostrar o resumo, lê a seção do `<nome>_base` (`config dump`) para saber se o usuário desligou a lixeira.

| Provedor | O que acontece | Fonte | Linha no resumo (**proposta**) |
|---|---|---|---|
| Google Drive | Vai para a lixeira, a menos que `use_trash = false`. O Google apaga de vez depois de 30 dias, e o que está na lixeira continua ocupando espaço | [rclone drive, Deleting files](https://rclone.org/drive/#deleting-files); [Google Drive Help](https://support.google.com/drive/answer/2375102) | `Os arquivos vão para a lixeira do Google Drive por 30 dias.` |
| OneDrive | Vai para a lixeira, a menos que `hard_delete = true` | [rclone onedrive, --onedrive-hard-delete](https://rclone.org/onedrive/#onedrive-hard-delete) | `Os arquivos vão para a lixeira do OneDrive.` |
| Dropbox | O rclone não descreve. O Dropbox guarda os apagados por 30, 180 ou 365 dias, conforme o plano | [Dropbox Help](https://help.dropbox.com/account-settings/data-retention-policy) | `O Dropbox guarda os arquivos apagados por um tempo, conforme o seu plano.` |
| S3 | Sem lixeira. Só fica uma versão antiga se o bucket tiver versionamento | [rclone s3, Versions](https://rclone.org/s3/#versions) | `Não tem lixeira: não dá para desfazer.` |
| Pasta Local | Sem lixeira: o rclone apaga os arquivos do disco. Eles **não** vão para a Lixeira do Windows nem do sistema | backend `local` | `Os arquivos são apagados do disco, sem passar pela lixeira. Não dá para desfazer.` |
| Lixeira desligada (`use_trash = false`, `hard_delete = true`) | Apaga de vez | as mesmas | `Não dá para desfazer.` |

Mesmo o que vai para a lixeira só volta cifrado: para abrir de novo, é preciso restaurar os arquivos pelo site do provedor e importar o cofre com a senha.

**Pasta Local, explícito:** hoje o app não cria nem importa cofres de Pasta Local (o wizard de criação filtra `LocalOnly` e `ConectarCofre` recusa `local_path`, ver 014). Quando a 014 existir, a segunda opção apaga do disco os arquivos cifrados do cofre que estão dentro da pasta escolhida. Isso vale para os arquivos e subpastas cujos nomes o crypt decifra. Arquivos de outra origem na mesma pasta ficam. A pasta em si fica, a menos que esteja vazia no fim. Nada passa pela lixeira.

#### Progresso, tempo e cancelamento

- Durante o apagamento, o diálogo mostra (**proposta**) `Apagando do {provedor}… {k} de {n} arquivos`, ou só `Apagando do {provedor}…` sem o resumo. O app conta as linhas de arquivo apagado do `rclone delete -v`. O texto exato da linha fica para conferir com o rclone real.
- Não há tempo limite total: um cofre grande pode levar horas, e o Google Drive limita a cerca de 2 arquivos por segundo ([rclone drive, Limitations](https://rclone.org/drive/#limitations)). Há um limite sem progresso (**proposta**: 5 min sem nenhum arquivo apagado). Passou disso, o app encerra o rclone e trata como falha parcial.
- Botão `Parar` (**proposta**): encerra o rclone. Também conta como falha parcial.
- Fechar o app durante o apagamento: o app pede confirmação antes de sair, no mesmo estilo da 028 (**proposta**). Se o usuário sair, o rclone é encerrado, o que foi apagado fica apagado, e vale a falha parcial.

#### Falha parcial

- O cofre só sai do app depois que o apagamento no provedor terminou por completo. Para saber, depois do `delete` o app roda `rclone size --json <remoto>:` de novo e exige 0 arquivos. Erro, tempo esgotado, `Parar` ou contagem diferente de 0 contam como falha parcial.
- Na falha parcial:
  - nada local é removido: o cofre, os remotos e o `vaults.json` ficam;
  - `vaults.json` ganha `exclusao_incompleta: true` no cofre, para o card avisar mesmo depois de fechar o app;
  - o card mostra (**proposta**) `Exclusão incompleta: ainda há arquivos no {provedor}.`;
  - destrancar continua possível, mas o conteúdo está pela metade;
  - a tela de editar abre já com a segunda opção do remover marcada e o resumo do que sobrou.
- Tentar de novo é seguro: o `delete` só apaga o que ainda existe. Mensagem (**proposta**): `Não excluiu tudo do {provedor}: {n} arquivos ficaram. Tente de novo.`
- Sem rede no meio: as frases da 027 (`sem conexão com o {provedor}`) entram no lugar do `{n}`.

#### Remoção local, nesta ordem

Vale para `Só deste computador` e para o fim de `Deste computador e do {provedor}`.

1. Confere de novo: sem processo do rclone para o cofre, sem modo só leitura (031) e sem pendentes no cache.
2. **Cache da VFS:** apaga `{cache}/vfs/<remoto>/` e `{cache}/vfsMeta/<remoto>/`. Sem pendentes, ali só há cópias de arquivos que já estão no provedor (ou que acabaram de ser apagados lá). Por que apagar:
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

#### O que `Só deste computador` NÃO faz

- **Nunca apaga nada no provedor.** Não roda `delete`, `purge`, `rmdir` nem nada que fale com a nuvem além do OAuth. Os arquivos cifrados continuam no Google Drive, OneDrive, Dropbox, S3 ou na pasta local, e dá para conectar de novo pelo `Importar Cofre Existente`, com a senha (e a senha 2, se for diferente).
- Não apaga nada da pasta da Pasta Local (os dados cifrados ficam nela).

#### O que nenhuma das duas opções faz

- Não revoga o acesso do app na conta do provedor. O rclone tem `config disconnect`, mas isso fica fora.
- Não apaga nada fora do que o crypt do cofre enxerga. Nunca roda `purge`, e nunca roda comando de apagar no remoto base.
- Não esvazia a lixeira do provedor.
- Não mexe em `vfs.json`: a configuração VFS é uma só, para todos os cofres.
- Não mexe no `Abrir ao ligar o computador` (atalho do sistema, 031).
- Não guarda cópia da senha em lugar nenhum. Com `Só deste computador`, se o usuário esqueceu a senha, os dados no provedor ficam ilegíveis para sempre. O corpo do diálogo diz que a senha vai ser necessária.

## Riscos

- **Apagar a conta inteira.** Os cofres criados pelo app ficam na raiz da conta. Um `purge`, ou qualquer comando de apagar no remoto base, apagaria todos os arquivos do usuário no provedor, não só os do cofre. A regra "só `delete` pelo crypt" é a proteção, e tem teste próprio.
- **Arquivos do próprio usuário na raiz.** Mesmo pelo crypt, alguns nomes comuns passam pela decifragem: em `standard`/base32, só nomes sem extensão, com `0-9`/`a-v` e 26, 52, 77… caracteres, e desses cerca de 1 em 256; em `obfuscate`, qualquer `<número>.<resto>`; em `off`, todo `.bin`. Por isso a raiz é sempre bloqueada. Enquanto a 035 não existir, nenhum cofre criado pelo app pode ser excluído do provedor pelo app.
- **Pasta com outros arquivos, detecção parcial.** A conferência olha só o primeiro nível, e um nome comum que passe pela decifragem não é percebido.
- **Cofres irmãos.** Dois cofres na mesma conta e no mesmo lugar, com a mesma senha e a mesma senha 2, decifram os nomes um do outro. O `delete` de um apagaria os arquivos do outro. O bloqueio por arquivos em comum cobre os cofres do `vaults.json`. Não cobre um cofre que só existe em outro computador ou fora do app.
- **Bloqueio demais.** Como o bloqueio é conservador, dois cofres do Drive na raiz bloqueiam um ao outro mesmo em contas diferentes (ver acima).
- **Senha diferente não é proteção.** No modo `standard`, cerca de 1 em 256 nomes de um cofre com outra senha passa pela decifragem do crypt e seria apagado. Em `off` e `obfuscate`, todos. Se o item 4 da condição 3 for liberado, esse risco passa a valer.
- **Segredos na comparação.** A comparação revela `password` e `password2` na memória. Eles não vão para o log, para a tela nem para o erro, e a comparação é em tempo constante, como a 030.
- **Sem volta.** S3, Pasta Local e provedores com a lixeira desligada perdem os dados na hora. Mesmo com lixeira, o Google apaga de vez em 30 dias.
- **Exclusão pela metade.** Um cofre com parte dos arquivos apagada continua destrancável e mostra um conteúdo incompleto. A marca `exclusao_incompleta` e a nova tentativa cobrem isso, mas o usuário pode ignorar.
- **Cota da API.** O `size` e o `delete` de um cofre grande no Google Drive gastam cota (032). Com o app compartilhado, isso pode ficar lento.
- **Renomear.** Ver "Riscos da decisão" em "Renomear".

## O que fica de fora

- Renomear os remotos no `rclone.conf` (ver pergunta 1).
- Mudar senha, provedor, pasta no provedor ou opções do crypt.
- Apagar no provedor e manter o cofre no app (opção descartada, ver "As duas opções").
- Esvaziar a lixeira do provedor, ou passar `--drive-use-trash=false` / `--onedrive-hard-delete`.
- `rclone config disconnect` para revogar o token ao remover.
- Mover os cofres que estão na raiz da conta para uma pasta própria.
- Editar `auto_montar` nesta tela (hoje ele não tem tela; a 015 cuida de auto-montar).
- Editar com o cofre destrancado.
- Desfazer uma remoção.

## Perguntas em aberto

1. **Renomear (Douglas):** a decisão aqui é mudar só o nome visto, sem tocar no `rclone.conf` nem no cache. Serve, ou o nome do remoto também deve mudar, com os riscos descritos acima?
2. **Confirmar (Douglas):** bloquear quando a conta e o caminho se cruzam e o crypt de um enxerga o outro, incluindo o item 4 da condição 3 (os dois em `standard`, senhas diferentes). Bloquear é o padrão recomendado pela UI e pelo time.
3. **Raiz da conta (Douglas):** os cofres novos devem passar a ficar numa pasta própria no provedor? Isso diminuiria o risco do apagamento e do "not recommended" do rclone. Seria outra demanda.
4. **Números do apagamento (Douglas):** os 2 min do resumo e os 5 min sem progresso são propostas.
5. **Frases marcadas como proposta (UI):** placeholder do secret, botões, `Cofre reconectado.`, título, corpo, campo e botão `Remover` do diálogo, a linha da Pasta Local, o resumo e as linhas de lixeira, o progresso e `Parar`, a falha parcial e o card de exclusão incompleta, o nome em uso e o cache ilegível.

Já decididas pela UI: o bloqueio sempre que o cofre está na raiz, ou com outros arquivos na pasta, e a frase `Não dá para excluir do {provedor}: este cofre está na raiz junto com outros arquivos. Remova só deste computador e apague pelo site do {provedor}.`; o bloqueio por arquivos em comum e a frase `Não dá para excluir: o cofre {outro} usa os mesmos arquivos no {provedor}.`; `Editar` também em `Não destrancou`, com `Reconectar` em destaque; duas opções, com `Só deste computador` marcada; o botão `Excluir do {provedor}`; sem pedir a senha de novo; a frase de bloqueio com `Destranque o cofre...`.

## Pronto quando

- [ ] Teste: `Editar` habilita com o cofre `Trancado` e em `Não destrancou` (com `Reconectar` em destaque); nos outros estados, desabilitado com `Tranque o cofre para editar.`. No modo só leitura (031), `Editar` e `Remover cofre` desabilitados, e o core recusa sem chamar o rclone.
- [ ] Teste com o rclone falso: `Reconectar` sem par chama `authorize <tipo>` e `config update -- <nome>_base token ...`; com par, `authorize drive <id> <secret>` e um `config update` com os três. Autorização que falha não chama `config update`.
- [ ] Teste: o app do Google só aparece para cofres do Google Drive; o secret gravado nunca aparece na tela nem no log.
- [ ] Teste: renomear muda só `nome` em `vaults.json` (e grava `remoto`); o `rclone.conf` falso e as pastas do cache ficam iguais; destrancar depois do nome novo monta `<remoto>:` com `--volname "RuntimeCrypto (<novo>)"` e (Linux) em `~/RuntimeCrypto/<novo>`.
- [ ] Teste: `vaults.json` antigo, sem `remoto`, abre com `remoto = nome` e não é regravado.
- [ ] Teste: nome novo igual ao `nome` ou ao `remoto` de outro cofre é recusado; voltar ao próprio nome antigo é aceito.
- [ ] Teste: remover, nas duas opções, com 1 e com 3 pendentes no cache recusa com a frase aprovada (singular e plural), sem chamar o rclone nem mexer em `vaults.json`; cache ilegível também recusa.
- [ ] Teste: `Só deste computador` apaga `vfs/<remoto>` e `vfsMeta/<remoto>` (e nada fora delas), chama `config delete <remoto>` e `config delete <remoto>_base`, tira de `vaults.json` e apaga a pasta de montagem vazia. Nenhum comando que fala com o provedor (`delete`, `purge`, `rmdir`, `rmdirs`, `deletefile`, `cleanup`) aparece em `chamadas.log`.
- [ ] Teste: crypt que não aponta para o base do cofre não é apagado; base usado por outra seção fica; falha no meio deixa o cofre na lista, e a segunda tentativa termina.
- [ ] Teste: `Deste computador e do {provedor}` chama, nesta ordem, `size --json <remoto>:`, `delete --rmdirs <remoto>:` (com `-v`), `size --json <remoto>:` e só então a remoção local. `purge` nunca aparece em `chamadas.log`, e nenhum comando de apagar recebe `<remoto>_base:` (só `rmdir <remoto>_base:<pasta>`, e nunca com pasta vazia).
- [ ] Teste: `delete` que falha, `Parar`, limite sem progresso e `size` final diferente de 0 deixam o cofre, os remotos e o cache; gravam `exclusao_incompleta: true`; o card mostra o aviso; a segunda tentativa que termina remove tudo.
- [ ] Teste: `size` que falha ou passa do tempo mostra o resumo sem números e não impede apagar.
- [ ] Teste do bloqueio por arquivos em comum, com dois cofres no `vaults.json` e o `rclone.conf` falso: mesmo tipo e os dois na raiz bloqueia; `Cofres` e `Cofres/A` bloqueia; `Cofres` e `Cofres2` não; `drive` e `onedrive` não; Drive com `team_drive` diferente nos dois não; Drive com `team_drive` só num deles bloqueia; OneDrive com `drive_id` diferente não; Dropbox sempre bloqueia quando os caminhos se cruzam. Bloqueado, a opção aparece desabilitada com a frase aprovada, `Só deste computador` funciona, e o core recusa o `delete` mesmo se a tela estiver velha.
- [ ] Teste da condição 3 (conta e caminho iguais, crypts do `rclone.conf` falso; com o item 4 em bloqueio, cada caso confere também o resultado da comparação em si):
  - tudo igual → enxerga, bloqueia;
  - `password` diferente → comparação diz "diferente";
  - `password2` diferente → "diferente"; os dois vazios → "igual" (sal padrão); um vazio e outro não → "diferente";
  - `filename_encryption` diferente (`standard` e `obfuscate`) → bloqueia pela regra do `obfuscate`;
  - `directory_name_encryption` diferente → "diferente";
  - `filename_encoding` diferente (`base32` e `base64`) → "diferente"; ausente e `base32` → "igual" (padrão);
  - `suffix` diferente com os dois em `standard` → não entra na comparação;
  - um dos dois em `off`, com senhas e `suffix` diferentes → bloqueia; o mesmo com `obfuscate`;
  - seção do crypt ausente, `password` que não se revela e `filename_encryption` desconhecido → bloqueia.
- [ ] Teste: nenhum valor de `password`/`password2`, revelado ou ofuscado, aparece em `runtimecrypto.log`, nas mensagens de erro ou na tela durante a comparação.
- [ ] Teste: a comparação não chama o rclone além do `config dump` (nenhum comando que fale com o provedor em `chamadas.log`).
- [ ] Teste: cofre com `remoto_base` na raiz (`<nome>_base:`, `<nome>_base:/`) tem `Deste computador e do {provedor}` desabilitada com a frase aprovada, mesmo sem nenhum outro cofre; `Só deste computador` funciona; o core recusa o `delete`, e nenhum `size`, `lsf` ou `delete` aparece em `chamadas.log`.
- [ ] Teste: cofre importado numa pasta, com o `lsf` do base mostrando uma entrada a mais que o `lsf` do crypt → bloqueia com a mesma frase; contagens iguais → libera; `lsf` com erro ou tempo esgotado → bloqueia.
- [ ] Teste com o crypt do rclone (nomes de exemplo, sem provedor): `Foto 2024.jpg`, `relatorio.pdf` e `notas` são recusados na decodificação em `standard`/base32; um nome de 26 caracteres só com `0-9a-v` chega à conferência do preenchimento; em `obfuscate`, `2024.relatorio.pdf` decodifica; em `off`, `x.bin` decodifica. O teste registra o comportamento em que a regra se apoia.
- [ ] Teste: a linha de lixeira segue o provedor e a seção do base (`use_trash = false` e `hard_delete = true` mostram `Não dá para desfazer.`).
- [ ] Teste da tela: o diálogo abre com `Só deste computador` marcada e `Os arquivos no {provedor} continuam lá.` abaixo da pergunta; ao marcar a outra opção, a linha some, o resumo aparece e o botão vira `Excluir do {provedor}`; nas duas, o botão só habilita com o nome digitado; não há campo de senha.
- [ ] Na tela, no Windows, com uma conta do Google Drive de teste: um cofre criado pelo app (na raiz) mostra a opção do provedor desabilitada com a frase da raiz. Um cofre importado numa pasta só dele: `Deste computador e do Google Drive` apaga os arquivos do cofre; os arquivos fora da pasta continuam; os do cofre aparecem na lixeira do Drive; o cofre sai do app. A mesma pasta com um arquivo comum a mais: bloqueia.
- [ ] Na tela, no Windows: `Reconectar` num cofre do Google Drive; trocar para o app próprio; renomear e destrancar; `Só deste computador` e conferir que os arquivos continuam no Drive e que `Importar Cofre Existente` abre de novo com a senha.

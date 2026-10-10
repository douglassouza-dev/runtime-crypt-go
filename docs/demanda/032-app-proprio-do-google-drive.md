# 032 — App próprio do Google (client ID e client secret) no Google Drive

- Estado: Aberta
- Risco: Médio · uso, segredos
- Onde: `internal/gui/wizards.go:DialogoNovoCofre`, `DialogoImportarCofre`; `internal/core/casos_de_uso.go:DadosNovoCofre`, `DadosConectarCofre`, `autorizar`, `criarRemotoBase`; `internal/core/oauth.go:GerenciadorOAuth.Iniciar`; `internal/core/criar_remoto.go:argsConfigCreate`; `internal/core/falhas_rclone.go:padroesFalhaRclone`
- Depende de: 027, 029; ADR-0006 para onde o `client_secret` fica guardado (ver "Onde fica o client secret")
- Vale só para cofres novos. Cofres que já existem trocam de app pela [033](033-editar-cofre.md) (tela de editar cofre).

## Contexto

### O que o app faz hoje

Todo cofre do Google Drive usa o app OAuth compartilhado do rclone, porque o programa nunca passa `client_id` nem `client_secret`:

- `GerenciadorOAuth.Iniciar` (`internal/core/oauth.go`) roda `rclone authorize <tipo>`, com `tipo` = `drive`, sem mais argumentos. Ele lê a URL `127.0.0.1` e o JSON do token na saída.
- `autorizar` (`internal/core/casos_de_uso.go`) espera a URL por até 5 s (`esperaURLOAuth`) e o token por até 2 min (`esperaTokenOAuth`).
- `criarRemotoBase` cria `<nome>_base` com `config create -- <nome>_base drive token <JSON>` (`argsConfigCreate`, demanda 029). O único parâmetro é `token`, sem `scope`, então vale o padrão do rclone (`drive`).
- Os dois wizards (`DialogoNovoCofre`, `DialogoImportarCofre`) só têm a escolha do provedor, senhas e nome. Não há campo avançado.
- Um `invalid_client` vindo do rclone hoje cai em `FalhaAutorizacao` pelo trecho `401 unauthorized` e a tela diz `autorização expirou`, o que é falso no momento da criação.
- O app não tem tela de editar cofre. A 031 já registra: "Remover e editar cofre não existem no app hoje."

### Por que isso pesa

1. **Limite dividido.** A doc do rclone diz que o `client_id` padrão "is shared between all the rclone users" e que "There is a global rate limit on the number of queries per second that each client_id can do set by Google. [...] The default Google quota is 10 transactions per second so it is recommended to stay under that number as if you use more than that, it will cause rclone to rate limit and make things slower." ([rclone, Making your own client_id](https://rclone.org/drive/#making-your-own-client-id))
2. **Como o limite aparece.** O Google conta cotas por minuto por projeto e por minuto por usuário por projeto. Quem passa recebe `403: User rate limit exceeded`, e outras checagens podem dar `429: Rate limit exceeded`. O cliente deve tentar de novo com espera exponencial ([Google Drive API, Usage limits](https://developers.google.com/workspace/drive/api/guides/limits)). Esperar é o que deixa a transferência lenta. Com um app próprio, a cota do projeto é só do usuário.
3. **O app compartilhado vai parar.** A doc do rclone diz: "This shared client_id is being retired and will stop working during 2026. To avoid interruption you must create and use your own client_id, so creating one is now required rather than merely recommended." O anúncio no fórum diz que o Google vai começar a cobrar pelas requisições do client ID padrão, "Later in 2026, following 90 days of notice", e que o rclone vai desligar esse client ID antes do fim do prazo ([fórum do rclone, ACTION REQUIRED](https://forum.rclone.org/t/google-drive-and-google-photos-users-action-required/54005)). Desde o commit [d03eb58](https://github.com/rclone/rclone/commit/d03eb5858618352f6cfd3e2026c05c6ea0f48515) (julho de 2026), o rclone avisa quem usa o client ID compartilhado.

Ou seja: além de "via expressa", o app próprio vai ser o único caminho para cofres do Google Drive quando o rclone desligar o compartilhado. Esta demanda continua opcional na tela, como a UI aprovou. Se o app próprio deve virar obrigatório, fica como pergunta em aberto.

### O que não se pode prometer

- **Nenhum número de velocidade.** O ganho depende de quanto o usuário usa e de quantos outros usam o app compartilhado no mesmo momento.
- **Limites que continuam.** A doc do rclone diz que o Drive "has quite a lot of rate limiting. This causes rclone to be limited to transferring about 2 files per second only", e o Google mantém o limite por usuário e os 750 GB de envio por dia ([rclone, Limitations](https://rclone.org/drive/#limitations); [Google, Usage limits](https://developers.google.com/workspace/drive/api/guides/limits)). O app próprio tira a disputa com outros usuários, não esses limites.
- **Cobrança.** Na página de limites, o Google diz que projetos criados a partir de 1º de maio de 2026 seguem as cotas novas e que passar da cota "is planned to incur charges to your Google Cloud billing account later in 2026". Há um limite diário de 400.000.000 unidades por projeto antes de cobrança. Os detalhes ainda não foram publicados.

## O que muda

### Na tela (aprovado pela UI)

- No passo do Google Drive dos dois wizards (`Criar Novo Cofre` e `Importar Cofre Existente`), aparece um link discreto: `Usar meu próprio app do Google`.
- Ao abrir o link, aparecem dois campos, `Client ID` e `Client secret`, e um link para o guia do rclone: https://rclone.org/drive/#making-your-own-client-id.
- Texto de ajuda: `Recomendado se você envia muitos arquivos. Evita os limites do app compartilhado.` Sem números.
- Com o link fechado, o fluxo fica exatamente como hoje: as mesmas chamadas ao rclone, com os mesmos argumentos.
- OneDrive, Dropbox, S3 e Pasta Local não mostram o link.
- ID ou secret recusados pelo Google aparecem no passo do OAuth como `Não deu para autorizar: o Google recusou esse app.`
- `Client secret` é campo de senha (`widget.NewPasswordEntry`).

### Validação, antes de chamar o rclone

- Os dois campos têm espaços e quebras de linha das pontas removidos.
- Os dois vazios: usa o app compartilhado, como hoje.
- Só um preenchido: não chama o rclone. O rclone e o fórum dizem que os dois andam juntos ("You need a client ID AND a client secret [...] They are a pair", [fórum](https://forum.rclone.org/t/config-problems-success-but-oauth-token-error/28363)).
- `Client ID` que não termina em `.apps.googleusercontent.com`: não chama o rclone. Esse é o formato dos exemplos da doc do rclone e da resposta do mantenedor no [fórum](https://forum.rclone.org/t/error-401-during-google-oauth-authentication-setting-up-google-drive/14832), não uma regra publicada pelo Google. O caso real do fórum era colar uma "API key" no lugar do client ID.
- Caractere de controle (por exemplo um byte nulo, caso real no [fórum](https://forum.rclone.org/t/trying-to-accomplish-backup-into-google-drive/3435)) ou espaço no meio: não chama o rclone.
- Frases da validação (**proposta**, falta aprovar):
  - só um campo: `Preencha o Client ID e o Client secret, ou nenhum dos dois.`
  - formato do ID: `O Client ID termina em .apps.googleusercontent.com.`
  - caractere inválido: `Tem um caractere que não vale aqui. Copie o valor de novo.`

### Como vai para o rclone

- Autorizar: `rclone authorize drive <client_id> <client_secret>`. A doc do `rclone authorize` aceita "a client_id and client_secret pair obtained from the remote service" como argumentos ([rclone authorize](https://rclone.org/commands/rclone_authorize/)). `GerenciadorOAuth.Iniciar` passa a receber os dois, opcionais.
- Criar o remoto: `config create -- <nome>_base drive client_id <id> client_secret <secret> token <JSON>` (chaves em ordem alfabética, com o `--` da 029). As opções `client_id` e `client_secret` são as do backend ([--drive-client-id](https://rclone.org/drive/#drive-client-id), [--drive-client-secret](https://rclone.org/drive/#drive-client-secret)).
- O token é emitido para aquele app. O rclone renova o token com o par gravado no remoto, então o par e o token têm de ir juntos no mesmo `config create`.
- O `scope` não muda: continua o padrão `drive`. O guia do usuário pede que o projeto tenha esse escopo.
- O log continua sem argumentos: `descreverComando` só escreve `config create` e `authorize`. Nem o ID nem o secret vão para o log nem para a tela.
- **Lista de processos.** O secret aparece nos argumentos do `authorize` e do `config create`, como o token já aparece hoje (demanda 005). Passar por variável de ambiente (`RCLONE_DRIVE_CLIENT_ID`, `RCLONE_DRIVE_CLIENT_SECRET`) é possível para o backend, mas a doc não diz se o `rclone authorize` lê essas variáveis. Isso fica para conferir com o rclone real antes do código. Se funcionar, esta demanda já usa ambiente; se não, a 005 cobre junto com o token.

### Erros no passo do OAuth

O que o Google e o rclone devolvem em cada caso (das fontes abaixo; o texto exato vem do rclone real e precisa ser conferido nos testes):

| Caso | Onde aparece | O que o rclone faz | O que o app faz (depois) |
|---|---|---|---|
| Client ID que não existe ou foi digitado errado | No navegador, página do Google: `Error 401: invalid_client`, `The OAuth client was not found.` ([fórum](https://forum.rclone.org/t/error-401-during-google-oauth-authentication-setting-up-google-drive/14832), [fórum](https://forum.rclone.org/t/new-installation-of-1-56-2-failing-with-invalid-client-on-macos/26899)) | Nada. Fica em `Waiting for code...` | Hoje espera os 2 min de `esperaTokenOAuth` e mostra `Autorização não concluída em 2m0s.` Ver pergunta 2 |
| Client secret errado | Depois do login, o rclone imprime `failed to get token: oauth2: cannot fetch token: 401 Unauthorized` com `"error": "invalid_client"` e `"error_description": "Unauthorized"` ([fórum](https://forum.rclone.org/t/config-problems-success-but-oauth-token-error/28363)) | Sai sem token | `Não deu para autorizar: o Google recusou esse app.` |
| Cliente apagado no projeto | `deleted_client`: "The OAuth client being used to make the request has been deleted" ([Google, OAuth para apps desktop](https://developers.google.com/identity/protocols/oauth2/native-app)) | No navegador ou na troca do código | A mesma frase, se chegar ao rclone |
| Usuário não está entre os test users, ou o login é recusado | `access_denied` ([Google, OAuth para apps desktop](https://developers.google.com/identity/protocols/oauth2/native-app)); a página de bloqueio fica no navegador | Fica esperando, ou sai sem token | Como hoje |
| Conta Workspace com o escopo bloqueado pelo admin | `admin_policy_enforced` (mesma fonte) | Como acima | Como hoje |

- Nova classe em `falhas_rclone.go`, `FalhaAppRecusado`, com os trechos `invalid_client`, `unauthorized_client` e `deleted_client`. Ela fica **antes** de `FalhaAutorizacao`, porque hoje `401 unauthorized` leva o `invalid_client` para `autorização expirou`.
- `FalhaAppRecusado` só vira a frase aprovada no passo do OAuth. Fora dele (montagem, seletor), a frase é a de `FalhaAutorizacao` até haver como trocar o app (ver "Cofres que já existem").
- Qualquer falha nesse passo desfaz o que foi criado, como já faz `CriacaoCofre.Desfazer` (006). Nenhum `<nome>_base` fica no `rclone.conf`.

### Projeto em modo Testing: o token vence em 7 dias

O Google diz: "A Google Cloud Platform project with an OAuth consent screen configured for an external user type and a publishing status of "Testing" is issued a refresh token expiring in 7 days" ([Google, Refresh token expiration](https://developers.google.com/identity/protocols/oauth2#expiration)). A doc do rclone repete: "Keeping the application in "Testing" will work as well, but the limitation is that any grants will expire after a week".

No app isso quer dizer:

- Sete dias depois de criar o cofre, montar e listar começam a falhar com `invalid_grant`. A tela mostra `autorização expirou` (027).
- **O app não tem como autorizar de novo** um cofre que já existe. Até a 033 (`Reconectar`), o cofre fica inutilizável depois de 7 dias.
- Por isso o guia do usuário manda publicar o app (`PUBLISH APP`, estado "In production"). O rclone diz que, para uso pessoal, dá para deixar o app sem verificação, aceitar a tela de aviso e publicar "to avoid the weekly grant expiry".
- Aviso perto dos campos (**proposta**, falta aprovar): `Publique o app no Google Cloud. Em modo de teste, o acesso vence em 7 dias.`
- O app não consegue saber se o projeto está em Testing antes de vencer. O token não traz essa informação.

### Cofres que já existem: fica para a 033

Esta demanda vale só para cofres novos, criados ou importados pelos wizards. O app não tem tela de editar cofre hoje (ver 031), e trocar o app de um cofre que já existe exige autorizar de novo. Isso fica na [033](033-editar-cofre.md), que cria a tela de editar cofre com os mesmos campos desta demanda e o botão `Reconectar`.

O que a 033 precisa saber daqui:

- Trocar só o par, sem token novo, não basta. O anúncio do rclone pede `rclone config reconnect` depois de pôr o par: "This part is important otherwise rclone won't actually use your new client_id/client_secret." Um token antigo com um par novo dá `invalid_client` na renovação ([fórum](https://forum.rclone.org/t/google-drive-wont-reconnect-after-resetting-credentials/32706)).
- O par e o token novo vão juntos para o `<nome>_base`. O remoto crypt e a senha não mudam.
- A validação, a classe `FalhaAppRecusado` e a frase do OAuth são as desta demanda.

### Onde fica o client secret

- Hoje vai para o `rclone.conf`, em texto puro, junto com o token (05-modelo-de-dados, "Onde ficam os segredos").
- O Google diz, sobre apps instalados, que o secret vai embutido no app e "In this context, the client secret is obviously not treated as a secret" ([Google, Installed applications](https://developers.google.com/identity/protocols/oauth2#installed)). Mesmo assim, quem tem o par usa a cota do projeto do usuário. Esta demanda o trata como segredo: campo de senha, fora do log, fora da tela depois de gravado.
- **Onde ele fica guardado depende do ADR-0006**, que ainda está pendente:
  - opções B e C (cifrar o `rclone.conf`): o secret fica protegido junto com o resto;
  - opções A e D: elas tratam só a senha do crypt; o secret (e o token) continuariam em texto puro no `rclone.conf`, a menos que o ADR seja estendido.
- Pergunta 2: esta demanda espera o ADR-0006, ou grava no `rclone.conf` como o token já é gravado e segue a decisão do ADR depois?

### Guia curto para o usuário

Resumo dos passos da doc do rclone ([Making your own client_id](https://rclone.org/drive/#making-your-own-client-id)). O link na tela leva para a doc completa, que é a que vale; o README pode ter este resumo.

1. Entrar no Google API Console com qualquer conta Google (não precisa ser a do Drive).
2. Escolher ou criar um projeto.
3. Em "ENABLE APIS AND SERVICES", procurar "Drive" e ativar a "Google Drive API".
4. Clicar em "Credentials" no painel da esquerda.
5. Configurar a tela de consentimento ("CONFIGURE CONSENT SCREEN", "Get started"): nome do app, e-mail de suporte, Audience "External", contato, aceitar os termos e "Create".
6. Em "Data Access", adicionar os escopos `https://www.googleapis.com/auth/docs`, `https://www.googleapis.com/auth/drive` e `https://www.googleapis.com/auth/drive.metadata.readonly` e salvar.
7. Em "Audience", adicionar a si mesmo como test user.
8. Em "Overview", "Create OAuth client", tipo "Desktop app". Anotar o client ID e o client secret.
9. Em "Audience", clicar em "PUBLISH APP" (se estiver cinza, preencher em "Branding" a página inicial e a política de privacidade). Sem isso, o acesso vence em 7 dias.
10. Colar o client ID e o client secret no app.

Na hora de autorizar, o Google mostra uma tela de "app não verificado". O rclone diz que, para uso pessoal (menos de 100 usuários), basta continuar.

## O que fica de fora

- Código. Esta demanda é só a doc.
- Mudar `scope` (por exemplo `drive.file`). Com `drive.file`, o rclone só vê o que ele mesmo criou ([rclone, Scopes](https://rclone.org/drive/#scopes)). Isso quebra "Importar Cofre Existente" para cofres criados fora do app.
- Ajustar `--drive-pacer-min-sleep` (padrão 100ms), `--drive-pacer-burst` (padrão 100) ou `--tpslimit` ([rclone, opções do drive](https://rclone.org/drive/#drive-pacer-min-sleep), [rclone, --tpslimit](https://rclone.org/docs/#tpslimit-float)). Pode virar outra demanda, mas só com medição.
- Service account.
- Cofres que já existem e a tela de editar cofre: [033](033-editar-cofre.md).
- Avisar quem ainda usa o app compartilhado. Pode virar outra demanda quando a data do desligamento for anunciada.
- OneDrive e Dropbox. O rclone tem o mesmo recurso, mas o caminho é outro:
  - **OneDrive:** o rclone diz que o Client ID padrão "are shared by all rclone users" e que criar um próprio ajuda "in case the default one does not work well for you. For example, you might see throttling." Exige registrar um app no Azure, com conta que pede telefone, endereço e cartão, e um secret que expira (o guia sugere 24 meses) ([rclone, OneDrive](https://rclone.org/onedrive/#getting-your-own-client-id-and-key)).
  - **Dropbox:** o App ID padrão também é "shared between all the rclone users". O app próprio é criado no Dropbox App console; o "App key" é o `client_id` e o "App secret" é o `client_secret` ([rclone, Dropbox](https://rclone.org/dropbox/#get-your-own-dropbox-app-id)). Para velocidade, a doc do Dropbox dá mais peso ao `--dropbox-batch-mode` do que ao app próprio.
  - Os dois podem ser demandas seguintes, com os mesmos campos e o mesmo `rclone authorize <tipo> <id> <secret>`. Os passos, os erros e o vencimento do secret são diferentes.

## Riscos

- O app compartilhado do rclone vai parar em 2026. Com o link fechado, um cofre novo do Drive vai deixar de funcionar quando isso acontecer. Esta demanda não resolve isso sozinha.
- Projeto em Testing: o cofre para de funcionar em 7 dias, e o app só vai poder autorizar de novo com a 033.
- Client ID errado não gera erro para o rclone: o usuário fica 2 minutos esperando, com a página de erro do Google no navegador.
- O rclone está mudando o fluxo do app compartilhado (aviso e pergunta no `rclone config`). O `rclone authorize drive` sem par pode passar a escrever um aviso a mais ou mudar a saída que `GerenciadorOAuth` lê. Precisa de teste com o rclone que o app distribui.
- Cobrança pelo Google no projeto do usuário, se ele passar da cota, a partir de quando o Google publicar as regras.
- O secret nos argumentos do processo, até a 005.

## Perguntas em aberto

1. **Client secret (Douglas):** espera o ADR-0006 ou grava no `rclone.conf` como o token e segue o ADR depois?
2. **Client ID errado (UI):** o Google mostra o erro no navegador e o rclone não fica sabendo. Depois dos 2 minutos, a tela mostra `Autorização não concluída em 2m0s.` (como hoje) ou a frase aprovada `Não deu para autorizar: o Google recusou esse app.` quando o par foi preenchido? Dá para encurtar essa espera?
3. **Obrigatório (Douglas):** com o desligamento do app compartilhado em 2026, o link continua opcional ou passa a ser o padrão para o Google Drive?
4. **Aviso dos 7 dias (UI):** a frase proposta sobre publicar o app entra, e onde?
5. **Frases de validação (UI):** as três frases marcadas como proposta.

## Pronto quando

- [ ] Teste com o rclone falso: com o link fechado, as chamadas em `chamadas.log` são idênticas às de hoje (`authorize drive` e `config create -- <nome>_base drive token ...`).
- [ ] Teste: com o par preenchido, o falso recebe `authorize drive <id> <secret>` e `config create -- <nome>_base drive client_id <id> client_secret <secret> token <JSON>`.
- [ ] Teste: só um campo preenchido, ID fora do formato e caractere de controle recusam sem chamar o rclone.
- [ ] Teste: o falso imprime a saída de `invalid_client` no `authorize` → `FalhaAppRecusado`, a tela mostra `Não deu para autorizar: o Google recusou esse app.`, e nenhum remoto fica no `rclone.conf` falso. A mesma saída fora do `authorize` não vira `autorização expirou` por engano nem a frase do OAuth.
- [ ] Teste: o log (`runtimecrypto.log`) e as frases da tela não contêm o ID nem o secret.
- [ ] Teste da tela: o link só aparece para Google Drive, nos dois wizards; abre os dois campos, o texto de ajuda e o link do guia; o secret é campo de senha.
- [ ] Conferido com o rclone real (versão que o app distribui): `rclone authorize drive <id> <secret>` imprime URL e token como `GerenciadorOAuth` espera; resposta registrada sobre variáveis de ambiente.
- [ ] Na tela, no Windows, com um projeto próprio em "In production": criar cofre, destrancar, gravar e listar; `rclone config show <nome>_base` mostra `client_id`.
- [ ] Na tela: client secret errado mostra a frase aprovada e não deixa remoto.
- [ ] README: o guia curto, o aviso dos 7 dias e o aviso de que o app compartilhado do rclone vai parar em 2026.

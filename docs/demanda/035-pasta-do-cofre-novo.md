# 035 — Criar cofre sempre grava na raiz da conta (`Pasta: /`)

- Estado: Aberta
- Risco: Médio · dados, uso
- Onde: `internal/core/casos_de_uso.go:CriarCofre` (`remotoBase = nomeBase + ":"`), `Interacao`; `internal/core/gerenciador.go:ListarDiretoriosRemoto`; `internal/gui/wizards.go:DialogoNovoCofre`; `internal/gui/seletor_pasta.go:DialogoSeletorPastaRemota`; `internal/gui/acoes.go:NovoCofre`, `TextoCofrePronto`
- Depende de: 006, 021, 023

## Contexto

`CriarCofre` aponta o crypt para `<nome>_base:`, a raiz da conta, e o diálogo de sucesso mostra `Pasta: /`. Isso causa três problemas:

- Os arquivos cifrados, com nomes ilegíveis, ficam misturados com os arquivos do usuário na raiz do Google Drive, OneDrive ou Dropbox. A doc do crypt chama isso de "not recommended" ([rclone crypt](https://rclone.org/crypt/#crypt-remote)).
- Dois cofres criados na mesma conta usam o mesmo lugar.
- Na 033, a raiz fica "dentro" de qualquer caminho. Por isso o bloqueio por arquivos em comum impede excluir do provedor qualquer cofre do mesmo provedor enquanto houver outro.

O seletor de pasta (021, 023) só aparece em "Importar Cofre Existente".

## O que muda

### Campo de pasta já preenchido (aprovado pela UI)

Na aba `2. Senha` do "Adicionar Cofre", abaixo de `Nome do cofre:`, aparece um campo novo (texto proposto):

- Rótulo: `Pasta no {provedor}:`
- Valor: o nome do cofre, convertido pela regra abaixo. Enquanto o usuário não mexe no campo, ele acompanha o que é digitado em `Nome do cofre:`. Depois que o usuário edita o campo, ele para de acompanhar.
- Dica embaixo: `O cofre guarda os arquivos só nesta pasta.`

Ao clicar em `Criar Cofre` (o botão que avança neste assistente), o cofre ganha a própria pasta. A raiz deixa de ser o padrão. Ordem nova: provedor → senha, confirmar, nome e pasta → `Criar Cofre` → autorizar no navegador → conferir ou criar a pasta → sucesso com `Pasta: /{pasta}`.

### Nome do cofre → nome da pasta

A regra é a mesma para os três provedores. O nome do cofre já passa pela regra da 006: letras, números, espaço e `_ . + @ -`, sem hífen no começo e sem espaço nas pontas. Sobram poucos casos, e uma regra só é mais fácil de testar e de explicar:

1. Parte do nome do cofre como está, mantendo maiúsculas, acentos e espaços.
2. Tira os pontos do fim. O OneDrive recusa nome que termina em ponto, e o Explorador do Windows também.
3. Se o resultado ficou vazio, ou é um nome reservado do Windows e do OneDrive (`CON`, `PRN`, `AUX`, `NUL`, `COM1`–`COM9`, `LPT1`–`LPT9`, sem diferenciar maiúsculas), vira `Cofre {resultado}`. Se estiver vazio, vira `Cofre`.
4. Corta em 255 caracteres, o limite de nome de pasta do OneDrive e do Dropbox.

Se o usuário digitar no campo um caractere que o provedor não aceita, o rclone já faz a troca pela codificação de cada backend ([rclone, Overview: encoding](https://rclone.org/overview/#encoding)). O app não repete essa troca. O campo aceita `/` para criar subpastas (`Cofres/Fotos`). Cada parte passa pelas regras 2 a 4.

### Conferir a pasta depois de autorizar

Com o token, antes de criar o crypt, o app lista a pasta com `rclone lsjson --max-depth 1 <nome>_base:<pasta>` e o tempo limite da 008:

| A pasta | O app faz |
|---|---|
| Não existe | Cria com `rclone mkdir <nome>_base:<pasta>` e segue. Sem pergunta. |
| Existe e está vazia | Usa a pasta e segue. Sem pergunta. |
| Existe e tem arquivos | Não usa a pasta sem perguntar. Abre o aviso abaixo. |
| A listagem falha | Para no passo, como a 027: `Não deu para conferir a pasta: {motivo}`. Desfaz os remotos (006). |

**Pasta com arquivos: oferecer o próximo nome livre (escolhido).** O aviso (texto proposto):

- Título: `Pasta em uso`
- Texto: `A pasta {pasta} já tem arquivos no {provedor}. Para não misturar, o cofre pode ficar em {pasta} (2).`
- Se a pasta for um cofre que já existe, a linha extra diz: `Se essa pasta já é um cofre, use Importar Cofre Existente.`
- Botões: `Usar {pasta} (2)`, em destaque, e `Cancelar`.

`(2)` é o primeiro de `(2)`, `(3)`… que não existe ou está vazio, até `(99)`. Depois disso, o aviso pede outro nome. `Cancelar` volta ao assistente com o que foi digitado, e o token não é pedido de novo enquanto o assistente está aberto. O remoto base fica na tentativa e é desfeito se o assistente fechar (006).

Por que essa opção e não só perguntar: o caminho comum é um clique e nunca mistura arquivos. Nada muda sem o usuário ver o nome novo. Perguntar "qual nome?" de volta faria o usuário inventar um nome que talvez também esteja ocupado. Usar a pasta mesmo assim fica de fora de propósito: com a mesma senha, dois cofres na mesma pasta decifram e apagam os arquivos um do outro (033, "cofres irmãos").

### Raiz de propósito (proposta: dá, com aviso)

O usuário pode apagar o campo para usar a raiz. O app não bloqueia, porque cofres antigos e cofres importados já estão na raiz. Mas pergunta antes de autorizar (texto proposto):

- Título: `Cofre na raiz`
- Texto: `O cofre vai ficar na raiz do {provedor}, junto com os seus arquivos. Assim ele não poderá ser excluído do {provedor} pelo app enquanto houver outro cofre do {provedor}.`
- Botões: `Usar a raiz` e `Voltar`, em destaque.

### Escolher no seletor (opcional)

Ao lado do campo, um link `Escolher no {provedor}…` abre, depois de autorizar, o seletor de pasta do importar com um botão `Nova pasta…` (`rclone mkdir`). A pasta escolhida passa pela mesma tabela de conferência. Erro: `Não deu para criar a pasta: {motivo}`.

### Relação com a 033

- Cofre na própria pasta: excluir do provedor roda `delete` pelo crypt e depois `rmdir` na pasta (033). Nada fora dela é tocado, e o risco de apagar a raiz da conta desaparece.
- Bloqueio por arquivos em comum (#50, na 033): dois cofres em `Fotos` e `Fotos (2)` não se bloqueiam, porque nenhum caminho está dentro do outro. Um cofre na raiz continua bloqueando todos os cofres do mesmo provedor, e é isso que o aviso de raiz explica.
- O `(2)` evita que o app crie dois cofres na mesma pasta, o caso de "cofres irmãos" que a 033 não consegue separar.

### Cofres que já existem

Nenhum impacto. `remoto_base` em `vaults.json` e o crypt no `rclone.conf` não mudam. Um cofre na raiz continua na raiz (`Pasta: /`). O importar continua igual.

## O que fica de fora

- Mover para uma pasta os arquivos de um cofre que já está na raiz.
- Detectar que a pasta com arquivos é um cofre e importar sozinho.
- S3 (034), em que o bucket faz o papel da pasta, e Pasta Local (014).
- Renomear ou apagar pasta no seletor.

## Pronto quando

- [ ] Teste da regra de nome: `Docs` → `Docs`; `fim.` → `fim`; `con` → `Cofre con`; `...` → `Cofre`; 300 caracteres → 255; `Cofres/Fotos.` → `Cofres/Fotos`.
- [ ] Teste em `internal/gui` com o driver de teste do Fyne: o campo `Pasta no {provedor}:` segue o nome até ser editado e depois para de seguir.
- [ ] Testes em `internal/core` com o rclone falso:
  - Pasta que não existe: roda `mkdir` e grava `remoto_base` como `<nome>_base:<pasta>`.
  - Pasta vazia: não roda `mkdir` nem pergunta.
  - Pasta com arquivos e `(2)` ocupada: oferece `(3)`. Aceitar grava `<nome>_base:<pasta> (3)`. `Cancelar` não grava nada.
  - Listagem que falha: para com `Não deu para conferir a pasta: …` e nem `<nome>_base` nem `<nome>` ficam no `rclone.conf`.
  - Campo vazio: só segue depois de `Usar a raiz` e grava `<nome>_base:`.
- [ ] Teste: um `vaults.json` com `remoto_base: "x_base:"` continua abrindo, listando e destrancando.
- [ ] Na tela, com uma conta real: criar `teste 035` sem mexer na pasta. O sucesso mostra `Pasta: /teste 035`, e os arquivos cifrados aparecem nessa pasta no site do provedor, não na raiz. Criar outro cofre com uma pasta que já tem um arquivo oferece `(2)`.

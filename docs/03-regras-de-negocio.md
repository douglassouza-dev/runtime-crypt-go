# 03 — Regras de negócio

Regras que o código aplica hoje. Cada linha aponta para o lugar onde a regra está. A coluna "Observação" registra onde a regra está, quando ela fica fora do `core`, e as lacunas.

## Cofre

| ID | Regra | Onde | Observação |
|---|---|---|---|
| RN-01 | O nome do cofre é a chave única. Dois cofres não podem ter o mesmo nome | `core/cofres.go:Adicionar` | A checagem acontece **depois** de o rclone criar os remotos. Ver [demanda 006](demanda/006-criacao-sobrescreve-remoto.md) |
| RN-02 | O nome não pode ser vazio | `gui/wizards.go:DialogoNovoCofre`, `DialogoImportarCofre` | A regra está na GUI, não no `core`. Não há regra de caracteres permitidos |
| RN-03 | Cada cofre usa dois remotos rclone: `<nome>_base` (provedor) e `<nome>` (crypt sobre o base) | `main.go:acaoNovoCofre`, `acaoImportarCofre` | A convenção está em `main.go` |
| RN-04 | Um cofre novo nasce com `auto_montar = false` | `core/cofres.go:Adicionar` | Nenhuma tela muda esse valor |
| RN-05 | `criado_em` usa o horário local, no formato `2006-01-02 15:04:05`, sem fuso | `core/cofres.go:Adicionar` | — |

## Senha

| ID | Regra | Onde | Observação |
|---|---|---|---|
| RN-06 | A senha de cofre novo tem pelo menos 8 caracteres | `gui/wizards.go:DialogoNovoCofre` | Conta bytes (`len`), não caracteres. Está na GUI |
| RN-07 | A senha e a confirmação precisam ser iguais | `gui/wizards.go:DialogoNovoCofre` | Está na GUI |
| RN-08 | Em cofre novo, `password2` (salt) é igual a `password` | `main.go:acaoNovoCofre` chama `CriarCrypt(nome, base, senha, senha, nil)` | Escolha fixa. O usuário não vê |
| RN-09 | Ao conectar cofre existente, `password2` vazio vira `password` | `gui/wizards.go:DialogoImportarCofre` e `core/gerenciador.go:CriarCrypt` | Regra duplicada nos dois lugares |
| RN-10 | As senhas vão para o `rclone.conf` ofuscadas por `rclone obscure` | `core/gerenciador.go:CriarCrypt` | Ofuscar não é cifrar. `rclone reveal` desfaz |
| RN-11 | A senha digitada ao destrancar fica no cache até trancar, falhar a montagem ou sair | `main.go:destravarCofre`, `travarCofre`; `core/gerenciador.go:Encerrar` | É a mesma senha que vai em `RCLONE_CONFIG_PASS` |
| RN-12 | Senha de destrancar não é conferida | — | Não existe regra. Com `rclone.conf` em claro, qualquer senha não vazia monta |

## Montagem

| ID | Regra | Onde | Observação |
|---|---|---|---|
| RN-13 | A letra é a primeira livre na ordem V, W, X, Y, Z, R, Q, ..., A | `core/constantes.go:LetrasPreferidas`, `core/montagem.go:ObterLetrasDisponiveis` | A lista inclui A, B e C. Fora do Windows, a lista volta vazia |
| RN-14 | Uma letra só pode ter uma montagem do programa | `core/montagem.go:MontarUnidade` | O mesmo cofre pode ser montado duas vezes em letras diferentes. Nada impede |
| RN-15 | A montagem só conta como pronta quando `X:\` existe, com limite de 45 s | `core/montagem.go:MontarUnidade` | Ao esgotar o tempo, o processo é morto |
| RN-16 | Toda montagem usa `--no-checksum` e `--no-modtime` | `core/montagem.go:MontarUnidade` | Fixo. Não aparece nas configurações |
| RN-17 | No Windows, a montagem usa `--network-mode --no-console` | `core/montagem.go:MontarUnidade` | — |
| RN-18 | Desmontar é: Interrupt, esperar 5 s, Kill, esperar 5 s, Kill, esperar 5 s | `core/montagem.go:DesmontarUnidade` | Informa sucesso em qualquer caso |
| RN-19 | Uma montagem conta como ativa enquanto `processoAtivo` devolve verdadeiro | `core/montagem.go:Status` | `processoAtivo` sempre devolve falso. Ver [demanda 001](demanda/001-rastreio-de-montagem.md) |
| RN-20 | Auto-montar só monta cofre com `auto_montar`, senha no cache e estado não montado | `main.go:autoMontarCofres` | Ao iniciar, o cache está vazio |

## VFS

| ID | Regra | Onde | Observação |
|---|---|---|---|
| RN-21 | Só as 13 chaves de `ConfiguracoesVfsPadrao` são aceitas | `core/vfs.go:Atualizar` | O valor não é validado |
| RN-22 | Valor vazio não apaga a chave. Mantém o valor anterior | `core/vfs.go:Atualizar`, `ConstruirArgs` | Não dá para limpar `cache_dir` depois de preenchido, a não ser com "Restaurar Padrões" |
| RN-23 | `cache_dir` vazio não gera flag | `core/vfs.go:ConstruirArgs` | O rclone usa o cache padrão dele |
| RN-24 | A configuração VFS vale só para a sessão | `core/vfs.go:NovoConfigVfs` | Não é gravada em disco |

## Provedores

| ID | Regra | Onde | Observação |
|---|---|---|---|
| RN-25 | Provedores: `drive`, `onedrive`, `dropbox` (OAuth), `s3` (campos), `local_path` (só conectar) | `core/provedores.go:Provedores` | Os `Campos` do S3 nunca são exibidos |
| RN-26 | "Novo cofre" esconde provedores `LocalOnly` | `gui/wizards.go:DialogoNovoCofre` | Deixa morto o ramo `local_path` de `main.go:acaoNovoCofre` |
| RN-27 | O OAuth espera o token por no máximo 120 s | `main.go:acaoNovoCofre`, `acaoImportarCofre` | O laço está em `main.go`, duplicado |

## Invariantes que o código não garante hoje

- Todo cofre em `vaults.json` tem um remoto crypt com o mesmo nome no `rclone.conf`. Não há checagem na leitura nem na escrita.
- Todo processo `rclone mount` iniciado pelo programa está no mapa de montagens até terminar. É quebrada por RN-19.
- Um cofre tem no máximo uma montagem ativa. Nada impede uma segunda.

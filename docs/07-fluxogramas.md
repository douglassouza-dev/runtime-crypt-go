# 07 — Fluxogramas

Cada fluxo descreve o que o código faz hoje. Os pontos com defeito aparecem em nós com a marca ⚠ e apontam para a demanda.

## Estados do cofre

### Regra alvo

Quatro estados visíveis: **desmontado**, **montando**, **montado**, **falhou**.
**montado** = processo `rclone mount` vivo **E** ponto de montagem existe (letra ou pasta).

```mermaid
stateDiagram-v2
    [*] --> desmontado
    desmontado --> montando: Destrancar com senha
    montando --> montado: processo vivo E ponto existe
    montando --> falhou: processo saiu, tempo esgotado ou erro ao iniciar
    montado --> falhou: processo morreu OU ponto sumiu
    montado --> desmontando: Trancar
    desmontando --> desmontado: processo terminou E ponto sumiu
    desmontando --> falhou: processo não terminou
    falhou --> montando: Tentar de novo
    falhou --> desmontado: Descartar, com processo encerrado
```

`desmontando` é um passo interno. Na tela pode aparecer como "montando" com outro texto ou ficar escondido. Isso fica para a [demanda 018](demanda/018-estados-do-cofre.md). Os quatro estados visíveis não mudam.

### Comportamento atual

```mermaid
stateDiagram-v2
    [*] --> Trancado
    Trancado --> Trancado: Destrancar, até 45 s sem sinal na tela
    Trancado --> Destrancado: MontarUnidade ok e mapa ainda tem a letra
    Destrancado --> Trancado: próxima chamada a Status, processoAtivo sempre falso
    Destrancado --> Trancado: Trancar
    note right of Destrancado
        Dura até a próxima atualização,
        no máximo 3 s. O rclone continua montado.
    end note
```

| Tela | O que mostra hoje | Fonte |
|---|---|---|
| Janela de cofres | "● Trancado" em vermelho ou "● Destrancado • X:\\" em verde. Botão "Destrancar" ou "Trancar" | `gui/janela_principal.go:criarCardCofre` |
| Wizard novo cofre | Nenhum estado. Depois de "Criar Cofre", nada aparece durante o OAuth até o diálogo "Autorização" ou um erro | `gui/wizards.go:DialogoNovoCofre`, `main.go:acaoNovoCofre` |
| Wizard conectar existente | Nenhum estado. O seletor mostra "🔄 Carregando pastas..." durante o `lsd` | `gui/seletor_pasta.go` |
| Ícone da bandeja | Nenhum estado. Tooltip fixo "RuntimeCrypto — Cofre Criptografado na Nuvem" | `tray/tray.go:aoIniciar` |

| Regra | Hoje |
|---|---|
| montado exige processo vivo | Tenta conferir com `processoAtivo`, que sempre devolve falso |
| montado exige ponto de montagem | Só confere durante a espera de 45 s |
| montando visível | Não existe |
| falhou visível | Só em diálogo. Some ao fechar |

## Criar cofre

```mermaid
flowchart TD
    A["Clique em Adicionar Cofre<br/>ou Novo Cofre na bandeja"] --> B["DialogoNovoCofre"]
    B --> C{"Provedor, nome,<br/>senha ≥ 8 e igual à confirmação?"}
    C -- não --> B
    C -- cancelar --> Z["Fim"]
    C -- sim --> D{"Provedor usa OAuth?"}
    D -- sim --> E["Fluxo OAuth"]
    E --> F{"Token em 120 s?"}
    F -- não --> F1["Diálogo Timeout"] --> Z
    F -- sim --> G["config create nome_base tipo token"]
    G --> G1{"ok?"}
    G1 -- não --> G2["Diálogo de erro"] --> Z
    G1 -- sim --> H
    D -- "não: S3" --> H0["⚠ Nenhum remoto base criado<br/>demanda 014"] --> H
    H["obscure senha<br/>config create nome crypt ..."] --> I{"ok?"}
    I -- não --> I1["⚠ Diálogo de erro<br/>nome_base fica órfão · demanda 006"] --> Z
    I -- sim --> J["Cofres.Adicionar"]
    J --> K{"Nome livre?"}
    K -- não --> K1["⚠ Erro, mas o rclone já sobrescreveu<br/>nome e nome_base · demanda 006"] --> Z
    K -- sim --> L["Grava vaults.json<br/>guarda senha no cache"] --> M["Diálogo Sucesso"] --> Z
```

## Conectar cofre existente

```mermaid
flowchart TD
    A["Clique em Importar Cofre Existente"] --> B["DialogoImportarCofre"]
    B --> C{"Provedor, nome e senha?"}
    C -- cancelar --> Z["Fim"]
    C -- não --> B
    C -- sim --> D{"Pasta Local?"}
    D -- sim --> D1["⚠ Mensagem e fim, TODO<br/>demanda 014"] --> Z
    D -- não --> E{"OAuth?"}
    E -- sim --> F["Fluxo OAuth e config create nome_base"]
    F --> F1{"Token em 120 s?"}
    F1 -- não --> F2["Diálogo Timeout"] --> Z
    F1 -- sim --> G
    E -- "não: S3" --> G0["⚠ Nenhum remoto base · demanda 014"] --> G
    G["DialogoSeletorPastaRemota<br/>rclone lsd, 30 s por pasta"] --> H{"Selecionou?"}
    H -- cancelar --> H1["config delete nome_base"] --> Z
    H -- sim --> I["obscure senhas<br/>config create nome crypt remote nome_base:pasta"]
    I --> J{"ok?"}
    J -- não --> J1["Diálogo de erro<br/>nome_base fica órfão"] --> Z
    J -- sim --> K["Cofres.Adicionar"] --> L["Diálogo Sucesso"] --> Z
```

## OAuth

```mermaid
flowchart TD
    A["GerenciadorOAuth.Iniciar"] --> B["Abortar OAuth anterior"]
    B --> C["rclone authorize tipo<br/>stdout e stderr no mesmo pipe"]
    C --> D{"Start ok?"}
    D -- não --> D1["Diálogo Falha ao iniciar autenticação"]
    D -- sim --> E["Goroutine lê blocos de 4096 bytes"]
    E --> F{"Linha com 127.0.0.1 ou localhost?"}
    F -- sim --> F1["Guarda URL"]
    E --> G{"Linha Paste the following?"}
    G -- sim --> G1["Junta linhas até End paste<br/>guarda JSON como token"]
    A --> H["Chamador dorme 1 s"]
    H --> I{"URL lida?"}
    I -- sim --> I1["AbrirNavegador URL<br/>diálogo Autorização"]
    I -- não --> J
    I1 --> J["Consulta token 1 vez por segundo<br/>até 120 vezes"]
    J --> K{"Concluido?"}
    K -- sim --> L["Valida JSON só no fluxo de cofre novo<br/>config create nome_base"]
    K -- "não em 120 s" --> M["Abortar e diálogo Timeout"]
```

Uma linha da saída pode ser cortada entre dois blocos de 4096 bytes. Nesse caso, a URL ou o token se perdem. O próprio `rclone authorize` também tenta abrir o navegador. Se o programa abre uma segunda aba é desconhecido.

## Destrancar (montar)

```mermaid
flowchart TD
    A["Clique em Destrancar"] --> B{"Senha no cache?"}
    B -- não --> C["DialogoSenha"]
    C --> C1{"Senha vazia ou cancelou?"}
    C1 -- sim --> Z["Fim"]
    C1 -- não --> D
    B -- sim --> D["Guarda senha no cache"]
    D --> E{"Windows?"}
    E -- "não" --> E1["⚠ Nenhuma letra disponível<br/>demanda 013"] --> X
    E -- sim --> F{"Letra livre?"}
    F -- não --> X["Limpa senha e mostra Erro ao Destrancar"] --> Z
    F -- sim --> G["Start: rclone mount nome: X: flags<br/>RCLONE_CONFIG_PASS=senha<br/>stdout e stderr descartados"]
    G --> G1{"Start ok?"}
    G1 -- não --> X
    G1 -- sim --> H["Espera: 200 ms × 1,5 até 1 s"]
    H --> I{"X:\\ existe?"}
    I -- sim --> J["Mapa letra → processo"]
    I -- não --> K{"⚠ ProcessState preenchido?<br/>nunca, sem Wait · demanda 003"}
    K -- não --> L{"Passaram 45 s?"}
    L -- não --> H
    L -- sim --> L1["Kill e Timeout"] --> X
    J --> M["Abre Explorer e diálogo Cofre Destrancado"]
    M --> N["ForcarAtualizacao → Status"]
    N --> O["⚠ processoAtivo falso → remove do mapa<br/>card volta a Trancado · demanda 001"] --> Z
```

## Trancar (desmontar)

```mermaid
flowchart TD
    A["Clique em Trancar"] --> B["ObterLetraPorRemoto nome"]
    B --> C{"Letra encontrada?"}
    C -- sim --> D["DesmontarUnidade letra"]
    C -- não --> D2["⚠ DesmontarUnidade nome<br/>usa o nome do cofre como letra · demanda 002"]
    D --> E{"Letra no mapa?"}
    D2 --> E
    E -- não --> E1["Erro: nenhuma montagem ativa"] --> L
    E -- sim --> F["Tentativa 1: Interrupt<br/>no Windows falha na hora"]
    F --> G{"Processo terminou em 5 s?"}
    G -- sim --> K
    G -- não --> H["Tentativas 2 e 3: Kill e espera 5 s"]
    H --> I{"Terminou?"}
    I -- sim --> K
    I -- "não após 3" --> K
    K["⚠ Remove do mapa e informa sucesso<br/>sem conferir · demanda 002"] --> L["Limpa senha do cache<br/>diálogo e atualização"]
```

## Auto-montar ao iniciar

```mermaid
flowchart TD
    A["main: espera 1,5 s"] --> B["Para cada cofre em ListarCofres"]
    B --> C{"AutoMontar?"}
    C -- não --> B
    C -- sim --> D{"⚠ TemSenha?<br/>cache vazio ao iniciar · demanda 015"}
    D -- não --> B
    D -- sim --> E{"Montado?"}
    E -- sim --> B
    E -- não --> F["MontarUnidade<br/>resultado ignorado"] --> B
```

## Sair

```mermaid
flowchart TD
    A["Sair na janela ou na bandeja"] --> B["Encerrar"]
    B --> C["DesmontarTodas"]
    C --> D{"⚠ Mapa vazio por causa de Status<br/>demanda 001"}
    D -- "vazio" --> E["Nada a desmontar<br/>rclone mount continua rodando"]
    D -- "com itens" --> F["DesmontarUnidade em cada letra"]
    E --> G["LimparTodas e Abortar OAuth"]
    F --> G
    G --> H["aplicacao.Quit<br/>systray.Quit não é chamado"]
```

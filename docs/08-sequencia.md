# 08 — Diagramas de sequência

Participantes: `U` usuário, `GUI` (Fyne), `Main` (`main.go`), `Core` (`internal/core`), `R` processos `rclone`, `Nav` navegador, `FS` sistema de arquivos e driver de montagem.

## Criar cofre (OAuth)

```mermaid
sequenceDiagram
    actor U as Usuário
    participant GUI as GUI Fyne
    participant Main as main.go
    participant Core as internal/core
    participant R as rclone
    participant Nav as Navegador

    U->>GUI: Adicionar Cofre
    GUI->>Main: CallbackNovoCofre
    Main->>GUI: DialogoNovoCofre (bloqueia)
    U->>GUI: provedor, senha, confirmação, nome
    GUI-->>Main: ResultadoNovoCofre
    Main->>Core: OAuth.Iniciar(exe, tipo)
    Core->>R: authorize tipo
    R-->>Core: stdout com URL 127.0.0.1
    Main->>Main: sleep 1 s
    Main->>Core: OAuth.ObterStatus
    Core-->>Main: URL
    Main->>Nav: AbrirNavegador(URL)
    U->>Nav: autoriza no provedor
    Nav->>R: redireciona para 127.0.0.1
    R-->>Core: Paste the following ... token ... End paste
    loop até 120 vezes, 1 s
        Main->>Core: OAuth.ObterStatus
    end
    Core-->>Main: Token
    Main->>Core: CriarRemoto(nome_base, tipo, token)
    Core->>R: config create nome_base tipo token JSON
    Note over Core,R: token visível na lista de processos
    R-->>Core: código 0
    Main->>Core: CriarCrypt(nome, nome_base:, senha, senha)
    Core->>R: obscure - (senha no stdin)
    R-->>Core: senha ofuscada
    Core->>R: obscure - (de novo, para password2)
    Core->>R: config create nome crypt remote ... password ... password2 ...
    Note over R: sobrescreve se nome já existir
    R-->>Core: código 0
    Main->>Core: Cofres.Adicionar
    Core->>Core: checa nome repetido e grava vaults.json
    Main->>Core: Senhas.Armazenar(nome, senha)
    Main->>GUI: DialogoMensagem Sucesso
```

## Conectar cofre existente

```mermaid
sequenceDiagram
    actor U as Usuário
    participant GUI as GUI Fyne
    participant Main as main.go
    participant Core as internal/core
    participant R as rclone

    U->>GUI: Importar Cofre Existente
    GUI->>Main: CallbackImportarCofre
    Main->>GUI: DialogoImportarCofre
    U->>GUI: provedor, password, password2, nome
    GUI-->>Main: ResultadoImportarCofre
    Note over Main,R: OAuth igual ao fluxo de criar cofre, até config create nome_base
    Main->>GUI: DialogoSeletorPastaRemota(nome_base)
    loop cada pasta aberta
        GUI->>Core: ListarDiretoriosRemoto(nome_base, caminho)
        Core->>R: lsd nome_base:caminho
        alt resposta em até 30 s
            R-->>Core: linhas
            Core-->>GUI: pastas ordenadas
        else erro ou 30 s
            Core->>R: Kill
            Core-->>GUI: nil, mostrado como nenhuma subpasta
        end
    end
    alt Cancelar
        GUI-->>Main: nil
        Main->>Core: RemoverRemoto(nome_base)
        Core->>R: config delete nome_base
    else Selecionar esta pasta
        GUI-->>Main: caminho
        Main->>Core: ImportarCrypt(nome, nome_base:caminho, senha, senha2)
        Core->>R: obscure, obscure, config create nome crypt ...
        Main->>Core: Cofres.Adicionar
        Main->>GUI: DialogoMensagem Sucesso
    end
```

## Destrancar (montar)

```mermaid
sequenceDiagram
    actor U as Usuário
    participant GUI as GUI Fyne
    participant Main as main.go
    participant Core as GerenciadorMontagem
    participant R as rclone mount
    participant FS as WinFsp / FUSE

    U->>GUI: Destrancar
    GUI->>Main: CallbackCofre(cofre)
    Main->>Core: ObterMontagens
    Core-->>Main: vazio, cofre não montado
    Main->>GUI: DialogoSenha
    U->>GUI: senha
    Main->>Main: Senhas.Armazenar
    Main->>Core: MontarUnidade(nome, letra vazia, senha, nil)
    Core->>Core: ObterLetrasDisponiveis, os.Stat em A:\ até Z:\
    Core->>R: Start mount nome: V: flags, env RCLONE_CONFIG_PASS
    Note over Core,R: stdout e stderr descartados, sem Wait
    R->>FS: registra V:
    loop até 45 s, intervalo crescente
        Core->>FS: os.Stat V:\
    end
    FS-->>Core: existe
    Core->>Core: montagens[V] = processo
    Core-->>Main: ok, V
    Main->>FS: explorer V:\
    Main->>GUI: DialogoMensagem Cofre Destrancado
    U->>GUI: OK
    Main->>GUI: ForcarAtualizacao
    GUI->>Core: ListarCofres, Status
    Core->>Core: processoAtivo, Signal nil dá erro
    Core->>Core: delete montagens[V]
    GUI-->>U: card mostra Trancado
    Note over R,FS: rclone continua montado em V:
```

## Trancar (desmontar)

```mermaid
sequenceDiagram
    actor U as Usuário
    participant GUI as GUI Fyne
    participant Main as main.go
    participant Core as GerenciadorMontagem
    participant R as rclone mount

    U->>GUI: Trancar
    GUI->>Main: CallbackCofre(cofre)
    Main->>Core: ObterLetraPorRemoto(nome)
    Core-->>Main: V
    Main->>Core: DesmontarUnidade(V)
    Core->>R: Signal Interrupt
    Note over Core,R: no Windows o Go recusa Interrupt
    alt terminou em 5 s
        R-->>Core: Wait retorna
    else não terminou
        Core->>R: Kill
        Core->>Core: espera 5 s, até 2 vezes
    end
    Core->>Core: delete montagens[V]
    Core-->>Main: ok, desmontada com sucesso
    Main->>Main: Senhas.Limpar(nome)
    Main->>GUI: DialogoMensagem Cofre Trancado
```

## OAuth

```mermaid
sequenceDiagram
    participant Main as main.go
    participant O as GerenciadorOAuth
    participant G as goroutine leitora
    participant R as rclone authorize

    Main->>O: Iniciar(exe, tipo)
    O->>O: Abortar anterior
    O->>R: Start authorize tipo, stdout e stderr no mesmo pipe
    O->>G: go ler saída
    O-->>Main: true
    loop blocos de 4096 bytes
        R-->>G: texto
        G->>G: procura URL e marcador Paste the following
        G->>O: url ou token sob mutex
    end
    R-->>G: EOF
    G->>R: Wait
    loop 120 vezes
        Main->>O: ObterStatus
        O-->>Main: URL, Token, Concluido
    end
    opt sem token
        Main->>O: Abortar
        O->>R: Kill e Wait
    end
```

## Auto-montar

```mermaid
sequenceDiagram
    participant Main as main.go
    participant Core as internal/core

    Main->>Main: sleep 1,5 s
    Main->>Core: ListarCofres
    Core-->>Main: cofres, TemSenha falso em todos
    loop cada cofre
        Main->>Main: AutoMontar e TemSenha e não Montado?
        Note over Main: TemSenha é falso ao iniciar, nada é montado
    end
```

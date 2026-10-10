.PHONY: todos compilar sistema-desconhecido windows windows-console linux macos limpar testes

NOME_BINARIO = runtime-crypt-go
FLAGS_PRODUCAO = -s -w
FLAGS_WINDOWS = -H windowsgui $(FLAGS_PRODUCAO)

# `make` sem alvo compila para o sistema atual.
ifeq ($(OS),Windows_NT)
  SISTEMA_ATUAL = windows
else
  UNAME_S := $(shell uname -s)
  ifeq ($(UNAME_S),Darwin)
    SISTEMA_ATUAL = macos
  else ifeq ($(UNAME_S),Linux)
    SISTEMA_ATUAL = linux
  else
    SISTEMA_ATUAL = sistema-desconhecido
  endif
endif

todos: compilar

compilar: $(SISTEMA_ATUAL)

sistema-desconhecido:
	$(error Sistema não reconhecido: $(UNAME_S). Use make windows, make linux ou make macos)

windows:
	go mod tidy
	go build -o $(NOME_BINARIO).exe -ldflags="$(FLAGS_WINDOWS)" .

windows-console:
	go mod tidy
	go build -o $(NOME_BINARIO).exe -ldflags="$(FLAGS_PRODUCAO)" .

linux:
	go mod tidy
	go build -o $(NOME_BINARIO) -ldflags="$(FLAGS_PRODUCAO)" .

macos:
	go mod tidy
	go build -o $(NOME_BINARIO) -ldflags="$(FLAGS_PRODUCAO)" .

testes:
	go test -race ./internal/core/...
	go test -race ./internal/gui/...

limpar:
	rm -f $(NOME_BINARIO) $(NOME_BINARIO).exe *.syso

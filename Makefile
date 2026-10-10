.PHONY: todos compilar windows windows-console linux macos limpar testes

NOME_BINARIO = runtime-crypt-go
FLAGS_PRODUCAO = -s -w
FLAGS_WINDOWS = -H windowsgui $(FLAGS_PRODUCAO)

todos: compilar

compilar: windows

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

package core

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// Testes da demanda 017: os casos de uso rodam com o rclone falso, sem GUI.

// interacaoFalsa faz o papel da tela.
type interacaoFalsa struct {
	urls     []string
	pastas   []string // remotos pedidos ao EscolherPasta
	caminho  string
	cancelar bool
}

func (i *interacaoFalsa) AbrirAutorizacao(url string) { i.urls = append(i.urls, url) }

func (i *interacaoFalsa) EscolherPasta(remotoBase, _ string) (string, bool) {
	i.pastas = append(i.pastas, remotoBase)
	return i.caminho, !i.cancelar
}

const tokenFalso = `{"access_token":"abc","token_type":"Bearer","expiry":"2030-01-01T00:00:00Z"}`

// esperasCurtas encurta as esperas do OAuth durante o teste.
func esperasCurtas(t *testing.T) {
	t.Helper()
	url, token, intervalo := esperaURLOAuth, esperaTokenOAuth, intervaloOAuth
	esperaURLOAuth, esperaTokenOAuth, intervaloOAuth = time.Second, 3*time.Second, 10*time.Millisecond
	t.Cleanup(func() { esperaURLOAuth, esperaTokenOAuth, intervaloOAuth = url, token, intervalo })
}

func lerConf(t *testing.T, caminho string) string {
	t.Helper()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(dados)
}

func TestCriarCofreDriveCriaBaseCryptEGrava(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	conf := confFalso(t, "")
	esperasCurtas(t)
	t.Setenv(envFalsoToken, tokenFalso)
	ui := &interacaoFalsa{}

	remoto, err := g.CriarCofre(DadosNovoCofre{
		Provedor: ObterProvedor("drive"), Nome: "fotos", Senha: "senha-forte", Confirmacao: "senha-forte",
	}, ui)

	if err != nil {
		t.Fatalf("CriarCofre: %v", err)
	}
	if remoto != "fotos_base:" {
		t.Errorf("remoto base = %q, quer fotos_base:", remoto)
	}
	if len(ui.urls) != 1 || !strings.Contains(ui.urls[0], "127.0.0.1") {
		t.Errorf("a URL do OAuth deveria ir para a tela uma vez: %v", ui.urls)
	}
	texto := lerConf(t, conf)
	for _, quer := range []string{"[fotos_base]", "type = drive", "[fotos]", "type = crypt", "remote = fotos_base:"} {
		if !strings.Contains(texto, quer) {
			t.Errorf("rclone.conf sem %q:\n%s", quer, texto)
		}
	}
	if !strings.Contains(texto, `"access_token":"abc"`) {
		t.Errorf("o token não chegou ao remoto base:\n%s", texto)
	}
	c := g.Cofres.Obter("fotos")
	if c == nil || c.RemotoBase != "fotos_base:" || c.ProvedorId != "drive" {
		t.Fatalf("vaults.json não tem o cofre: %+v", c)
	}
	if g.Senhas.Obter("fotos") != "senha-forte" {
		t.Error("a senha deveria ficar na sessão")
	}
}

func TestCriarCofreRegrasDoWizardNoCore(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	confFalso(t, "")
	casos := map[string]DadosNovoCofre{
		"sem provedor": {Nome: "a", Senha: "12345678", Confirmacao: "12345678"},
		"sem nome":     {Provedor: ObterProvedor("drive"), Senha: "12345678", Confirmacao: "12345678"},
		"senha curta":  {Provedor: ObterProvedor("drive"), Nome: "a", Senha: "1234567", Confirmacao: "1234567"},
		"senha difere": {Provedor: ObterProvedor("drive"), Nome: "a", Senha: "12345678", Confirmacao: "12345679"},
	}
	for nome, d := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := g.CriarCofre(d, &interacaoFalsa{}); err == nil {
				t.Fatal("deveria recusar")
			}
		})
	}
	if len(f.chamadas()) != 0 {
		t.Errorf("dado inválido não deveria chamar o rclone: %+v", f.chamadas())
	}
}

func TestCriarCofreFalhaNoCryptDesfazTudo(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	conf := confFalso(t, "")
	esperasCurtas(t)
	t.Setenv(envFalsoToken, tokenFalso)
	t.Setenv(envFalsoFalhaTipo, "crypt")

	_, err := g.CriarCofre(DadosNovoCofre{
		Provedor: ObterProvedor("drive"), Nome: "fotos", Senha: "senha-forte", Confirmacao: "senha-forte",
	}, &interacaoFalsa{})

	if err == nil {
		t.Fatal("deveria falhar")
	}
	if texto := lerConf(t, conf); strings.Contains(texto, "fotos") {
		t.Errorf("sobrou remoto no rclone.conf:\n%s", texto)
	}
	if g.Cofres.Obter("fotos") != nil || g.Senhas.Obter("fotos") != "" {
		t.Error("não deveria gravar cofre nem senha")
	}
}

func TestCriarCofreOAuthSemTokenEsgotaETiraORemoto(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	conf := confFalso(t, "")
	esperasCurtas(t)
	esperaTokenOAuth = 500 * time.Millisecond

	_, err := g.CriarCofre(DadosNovoCofre{
		Provedor: ObterProvedor("drive"), Nome: "fotos", Senha: "senha-forte", Confirmacao: "senha-forte",
	}, &interacaoFalsa{})

	if err == nil || !strings.Contains(err.Error(), "Autorização não concluída") {
		t.Fatalf("erro = %v", err)
	}
	if houveConfigCreate(f) {
		t.Error("sem token não deveria criar remoto")
	}
	if texto := lerConf(t, conf); strings.Contains(texto, "fotos") {
		t.Errorf("rclone.conf mudou:\n%s", texto)
	}
}

func TestCriarCofrePastaLocal(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	conf := confFalso(t, "")

	remoto, err := g.CriarCofre(DadosNovoCofre{
		Provedor: ObterProvedor("local_path"), Nome: "docs", Senha: "senha-forte", Confirmacao: "senha-forte",
	}, &interacaoFalsa{})

	if err != nil {
		t.Fatal(err)
	}
	texto := lerConf(t, conf)
	if remoto != "docs_base:" || !strings.Contains(texto, "[docs_base]\ntype = local") {
		t.Errorf("remoto = %q, conf:\n%s", remoto, texto)
	}
}

func TestConectarCofreUsaPastaEscolhidaESenha2Padrao(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	conf := confFalso(t, "")
	esperasCurtas(t)
	t.Setenv(envFalsoToken, tokenFalso)
	ui := &interacaoFalsa{caminho: "Cofres/fotos"}

	remoto, err := g.ConectarCofre(DadosConectarCofre{
		Provedor: ObterProvedor("drive"), Nome: "fotos", Senha: "senha-antiga",
	}, ui)

	if err != nil {
		t.Fatalf("ConectarCofre: %v", err)
	}
	if remoto != "fotos_base:Cofres/fotos" {
		t.Errorf("remoto = %q", remoto)
	}
	if len(ui.pastas) != 1 || ui.pastas[0] != "fotos_base" {
		t.Errorf("EscolherPasta deveria receber o remoto base: %v", ui.pastas)
	}
	if texto := lerConf(t, conf); !strings.Contains(texto, "remote = fotos_base:Cofres/fotos") {
		t.Errorf("crypt não aponta para a pasta escolhida:\n%s", texto)
	}
	// password2 padrão = a própria senha: as duas obscure recebem o mesmo texto.
	var obscurecidas []string
	for _, c := range f.chamadas() {
		if len(c.Args) > 0 && c.Args[0] == "obscure" {
			obscurecidas = append(obscurecidas, c.Stdin+strings.Join(c.Args[1:], ""))
		}
	}
	if len(obscurecidas) != 2 || obscurecidas[0] != obscurecidas[1] {
		t.Errorf("password2 deveria ser a senha: %q", obscurecidas)
	}
	if g.Cofres.Obter("fotos") == nil {
		t.Error("cofre não gravado")
	}
}

func TestConectarCofreCanceladoNaPastaDesfaz(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	conf := confFalso(t, "")
	esperasCurtas(t)
	t.Setenv(envFalsoToken, tokenFalso)

	_, err := g.ConectarCofre(DadosConectarCofre{
		Provedor: ObterProvedor("drive"), Nome: "fotos", Senha: "x",
	}, &interacaoFalsa{cancelar: true})

	if !errors.Is(err, ErrCancelado) {
		t.Fatalf("erro = %v, quer ErrCancelado", err)
	}
	if texto := lerConf(t, conf); strings.Contains(texto, "fotos") {
		t.Errorf("o remoto base deveria ter saído:\n%s", texto)
	}
	if g.Cofres.Obter("fotos") != nil {
		t.Error("não deveria gravar o cofre")
	}
}

func TestConectarCofreSenhaVaziaRecusada(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	if _, err := g.ConectarCofre(DadosConectarCofre{Provedor: ObterProvedor("drive"), Nome: "a"}, &interacaoFalsa{}); err == nil {
		t.Fatal("deveria recusar senha vazia")
	}
	if len(f.chamadas()) != 0 {
		t.Error("não deveria chamar o rclone")
	}
}

func TestDestrancarCanceladoNaoMonta(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	if _, err := g.Destrancar("fotos", func() (string, bool) { return "", false }); !errors.Is(err, ErrCancelado) {
		t.Fatalf("erro = %v", err)
	}
	if len(f.chamadas()) != 0 {
		t.Error("não deveria chamar o rclone")
	}
}

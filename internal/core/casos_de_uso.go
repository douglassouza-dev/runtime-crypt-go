package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Casos de uso do programa (demanda 017). Cada um recebe dados e devolve
// resultado ou erro, sem diálogo. O que precisa do usuário no meio do caminho
// (abrir o navegador, escolher a pasta) entra pela interface Interacao, que a
// GUI implementa.

// Interacao é o que os casos de uso pedem a quem está na tela.
type Interacao interface {
	// AbrirAutorizacao recebe a URL do OAuth. A GUI abre o navegador e avisa
	// o usuário. Não espera o login terminar.
	AbrirAutorizacao(url string)
	// EscolherPasta mostra as pastas do remoto e devolve a escolhida. ok=false
	// quando o usuário cancela.
	EscolherPasta(remotoBase string, tituloProvedor string) (caminho string, ok bool)
	// Passo avisa em que passo o caso de uso entrou (demanda 018).
	Passo(p Passo)
}

// Passo é uma etapa de CriarCofre/ConectarCofre (demanda 018).
type Passo string

const (
	PassoAutorizando   Passo = "autorizando"
	PassoCriandoRemoto Passo = "criando remoto"
	PassoGravando      Passo = "gravando"
)

// ErroPasso é o erro de um passo de CriarCofre/ConectarCofre: a tela diz em
// que passo parou.
type ErroPasso struct {
	Passo Passo
	Err   error
}

func (e *ErroPasso) Error() string { return e.Err.Error() }
func (e *ErroPasso) Unwrap() error { return e.Err }

func noPasso(p Passo, err error) error {
	if err == nil {
		return nil
	}
	return &ErroPasso{Passo: p, Err: err}
}

// TamanhoMinimoSenha é o mínimo de caracteres da senha de um cofre novo.
const TamanhoMinimoSenha = 8

// DadosNovoCofre é o que o wizard de novo cofre coleta.
type DadosNovoCofre struct {
	Provedor    *Provedor
	Nome        string
	Senha       string
	Confirmacao string
}

// ValidarNovoCofre aplica as regras do wizard de novo cofre.
func ValidarNovoCofre(d DadosNovoCofre) error {
	if d.Provedor == nil {
		return errors.New("Selecione um provedor primeiro.")
	}
	if d.Nome == "" {
		return errors.New("Informe um nome para o cofre.")
	}
	if len(d.Senha) < TamanhoMinimoSenha {
		return fmt.Errorf("A senha deve ter pelo menos %d caracteres.", TamanhoMinimoSenha)
	}
	if d.Senha != d.Confirmacao {
		return errors.New("As senhas não coincidem.")
	}
	return nil
}

// DadosConectarCofre é o que o wizard de conectar cofre existente coleta.
// Senha2 vazia usa a própria Senha (password2 padrão do rclone crypt).
type DadosConectarCofre struct {
	Provedor *Provedor
	Nome     string
	Senha    string
	Senha2   string
}

// ValidarConectarCofre aplica as regras do wizard de conectar cofre.
func ValidarConectarCofre(d DadosConectarCofre) error {
	if d.Provedor == nil {
		return errors.New("Selecione um provedor primeiro.")
	}
	if d.Nome == "" {
		return errors.New("Informe um nome para o cofre.")
	}
	if d.Senha == "" {
		return errors.New("Informe a senha do cofre.")
	}
	return nil
}

// ErrCancelado é devolvido quando o usuário desiste no meio (ex.: fecha o
// seletor de pasta).
var ErrCancelado = errors.New("cancelado pelo usuario")

// Esperas do OAuth. Variáveis só para os testes encurtarem.
var (
	// esperaURLOAuth é quanto se espera o rclone authorize imprimir a URL.
	esperaURLOAuth = 5 * time.Second
	// esperaTokenOAuth é quanto se espera o usuário terminar o login.
	esperaTokenOAuth = 2 * time.Minute
	intervaloOAuth   = 200 * time.Millisecond
)

// autorizar roda o OAuth do provedor e devolve o token JSON.
func (g *GerenciadorRClone) autorizar(prov *Provedor, ui Interacao) (string, error) {
	if !g.OAuth.Iniciar(g.Executavel, prov.Id) {
		return "", errors.New("Falha ao iniciar autenticação.")
	}
	defer g.OAuth.Abortar()

	urlAberta := false
	inicio := time.Now()
	for {
		st := g.OAuth.ObterStatus()
		if !urlAberta && st.URL != "" {
			ui.AbrirAutorizacao(st.URL)
			urlAberta = true
		}
		if st.Concluido {
			var token map[string]interface{}
			if err := json.Unmarshal([]byte(st.Token), &token); err != nil {
				return "", errors.New("Token OAuth inválido.")
			}
			return st.Token, nil
		}
		if st.Falha != nil {
			return "", st.Falha
		}
		if st.Erro != "" {
			return "", fmt.Errorf("Autorização não concluída: %s", st.Erro)
		}
		if !urlAberta && time.Since(inicio) > esperaURLOAuth {
			urlAberta = true // segue esperando o token sem URL
		}
		if time.Since(inicio) > esperaTokenOAuth {
			return "", fmt.Errorf("Autorização não concluída em %s.", esperaTokenOAuth)
		}
		time.Sleep(intervaloOAuth)
	}
}

// criarRemotoBase cria `<nome>_base` conforme o provedor: OAuth com token,
// Pasta Local como remoto local. Devolve o nome do remoto base.
func (g *GerenciadorRClone) criarRemotoBase(c *CriacaoCofre, prov *Provedor, ui Interacao) (string, error) {
	nomeBase := NomeRemotoBase(c.Nome())
	switch {
	case prov.OAuth:
		ui.Passo(PassoAutorizando)
		token, err := g.autorizar(prov, ui)
		if err != nil {
			return "", noPasso(PassoAutorizando, err)
		}
		ui.Passo(PassoCriandoRemoto)
		if err := c.CriarRemoto(nomeBase, prov.Id, map[string]string{"token": token}); err != nil {
			return "", noPasso(PassoCriandoRemoto, err)
		}
	case prov.Id == "local_path":
		ui.Passo(PassoCriandoRemoto)
		if err := c.CriarRemoto(nomeBase, "local", map[string]string{"remote": ""}); err != nil {
			return "", noPasso(PassoCriandoRemoto, err)
		}
	}
	return nomeBase, nil
}

// CriarCofre cria um cofre novo: confere o nome, autoriza (OAuth), cria o
// remoto base, o crypt e grava em vaults.json. Qualquer falha no meio desfaz
// os remotos criados (006). No fim, a senha fica na sessão.
func (g *GerenciadorRClone) CriarCofre(d DadosNovoCofre, ui Interacao) (remotoBase string, err error) {
	if err := ValidarNovoCofre(d); err != nil {
		return "", err
	}
	criacao, err := g.IniciarCriacaoCofre(d.Nome)
	if err != nil {
		return "", err
	}
	defer criacao.Desfazer()

	nomeBase, err := g.criarRemotoBase(criacao, d.Provedor, ui)
	if err != nil {
		return "", err
	}
	remotoBase = nomeBase + ":"
	ui.Passo(PassoCriandoRemoto)
	if err := criacao.CriarCrypt(remotoBase, d.Senha, d.Senha, nil); err != nil {
		return "", noPasso(PassoCriandoRemoto, err)
	}
	ui.Passo(PassoGravando)
	if ok, msg := criacao.Concluir(d.Provedor.Id, d.Provedor.Nome, remotoBase); !ok {
		return "", noPasso(PassoGravando, errors.New(msg))
	}
	g.Senhas.Armazenar(d.Nome, d.Senha)
	return remotoBase, nil
}

// ConectarCofre conecta um cofre crypt que já existe na nuvem: autoriza,
// cria o remoto base, pede a pasta ao usuário e cria o crypt sobre ela.
func (g *GerenciadorRClone) ConectarCofre(d DadosConectarCofre, ui Interacao) (remotoBase string, err error) {
	if err := ValidarConectarCofre(d); err != nil {
		return "", err
	}
	if d.Provedor.Id == "local_path" {
		// Comportamento de hoje: Pasta Local ainda não conecta (demanda 014).
		return "", errors.New("Selecione a pasta no explorador.")
	}
	senha2 := d.Senha2
	if senha2 == "" {
		senha2 = d.Senha
	}

	criacao, err := g.IniciarCriacaoCofre(d.Nome)
	if err != nil {
		return "", err
	}
	defer criacao.Desfazer()

	nomeBase, err := g.criarRemotoBase(criacao, d.Provedor, ui)
	if err != nil {
		return "", err
	}
	caminho, ok := ui.EscolherPasta(nomeBase, d.Provedor.Nome)
	if !ok {
		return "", ErrCancelado
	}
	remotoBase = nomeBase + ":" + caminho

	ui.Passo(PassoCriandoRemoto)
	if err := criacao.CriarCrypt(remotoBase, d.Senha, senha2, nil); err != nil {
		return "", noPasso(PassoCriandoRemoto, err)
	}
	ui.Passo(PassoGravando)
	if ok, msg := criacao.Concluir(d.Provedor.Id, d.Provedor.Nome, remotoBase); !ok {
		return "", noPasso(PassoGravando, errors.New(msg))
	}
	g.Senhas.Armazenar(d.Nome, d.Senha)
	return remotoBase, nil
}

// Destrancar monta o cofre. Usa a senha da sessão se houver; senão chama
// pedirSenha (ok=false: o usuário cancelou, ErrCancelado). Se a montagem
// falha, a senha sai da sessão. Devolve o ponto de montagem: `V:\` no
// Windows, a pasta em Linux e macOS (demanda 013).
//
// Demanda 030: a senha pedida é conferida antes de tudo. Errada, volta
// ErrSenhaErrada sem guardar a senha nem iniciar o rclone; o cofre continua
// trancado. A senha da sessão não é conferida de novo: ela veio de uma
// conferência que passou ou da criação/conexão do cofre.
func (g *GerenciadorRClone) Destrancar(nome string, pedirSenha func() (string, bool)) (string, error) {
	senha := g.Senhas.Obter(nome)
	if senha == "" {
		var ok bool
		senha, ok = pedirSenha()
		if !ok || senha == "" {
			return "", ErrCancelado
		}
		if err := g.ConferirSenha(nome, senha); err != nil {
			return "", err
		}
	}
	g.Senhas.Armazenar(nome, senha)
	ok, msg, letra := g.Montagens.MontarUnidade(nome, "", senha, nil)
	if !ok || letra == "" {
		g.Senhas.Limpar(nome)
		if e := g.Montagens.falhaRclone(nome); e != nil {
			return "", e
		}
		return "", errors.New(msg)
	}
	return g.Montagens.caminhoPonto(letra), nil
}

// EstaMontado diz se o cofre está montado agora.
func (g *GerenciadorRClone) EstaMontado(nome string) bool {
	return g.EstadoDoCofre(nome).Estado == EstadoMontado
}

// EstadoDoCofre devolve o estado atual do cofre nome (demanda 018).
func (g *GerenciadorRClone) EstadoDoCofre(nome string) EstadoRemoto {
	if e, ok := g.Montagens.EstadosPorRemoto()[nome]; ok {
		return e
	}
	return EstadoRemoto{Estado: EstadoDesmontado}
}

// AutoMontar monta os cofres marcados para auto-montagem que têm senha na
// sessão e ainda não estão montados. Devolve um erro por cofre que falhou.
func (g *GerenciadorRClone) AutoMontar() map[string]error {
	falhas := map[string]error{}
	for _, cofre := range g.ListarCofres() {
		if !cofre.AutoMontar || !cofre.TemSenha || cofre.Estado == EstadoMontado || cofre.Estado == EstadoMontando {
			continue
		}
		senha := g.Senhas.Obter(cofre.Nome)
		if senha == "" {
			continue
		}
		if ok, msg, _ := g.Montagens.MontarUnidade(cofre.Nome, "", senha, nil); !ok {
			falhas[cofre.Nome] = errors.New(msg)
		}
	}
	return falhas
}

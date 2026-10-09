package core

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Testes da demanda 008: toda chamada ao rclone tem tempo limite.
// O rclone falso dorme 10 min; os limites são encurtados para 1 s.

const limiteTeste = time.Second

// encurtarLimites troca todos os limites por d e devolve os originais no fim.
func encurtarLimites(t *testing.T, d time.Duration) {
	t.Helper()
	antes := []time.Duration{limiteVersao, limiteObscure, limiteConfigLocal, limiteConfigCreate, limiteListagem, limiteAuthorize}
	limiteVersao, limiteObscure, limiteConfigLocal, limiteConfigCreate, limiteListagem, limiteAuthorize = d, d, d, d, d, d
	t.Cleanup(func() {
		limiteVersao, limiteObscure, limiteConfigLocal, limiteConfigCreate, limiteListagem, limiteAuthorize =
			antes[0], antes[1], antes[2], antes[3], antes[4], antes[5]
	})
}

// semProcessoVivo falha se algum processo registrado pelo falso ainda existe.
// No Unix um zumbi conta como vivo, então isto também confere o Wait.
func semProcessoVivo(t *testing.T, f *rcloneFalso) {
	t.Helper()
	cs := f.chamadas()
	if len(cs) == 0 {
		t.Fatal("o rclone falso não foi chamado")
	}
	for _, c := range cs {
		if processoVivoNoSO(c.Pid) {
			t.Errorf("processo %d (%v) continua vivo depois do tempo esgotado", c.Pid, c.Args)
		}
	}
}

func dentroDoLimite(t *testing.T, inicio time.Time) {
	t.Helper()
	if d := time.Since(inicio); d > limiteTeste+time.Second {
		t.Errorf("demorou %v; o limite é %v + 1 s", d, limiteTeste)
	}
}

func eTempoEsgotado(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrTempoEsgotado) || !strings.Contains(err.Error(), "tempo esgotado") {
		t.Errorf("esperava erro de tempo esgotado, veio %v", err)
	}
}

func preparaDorminhoco(t *testing.T) (*GerenciadorRClone, *rcloneFalso) {
	t.Helper()
	g, f := novoGerenciadorFalso(t)
	encurtarLimites(t, limiteTeste)
	f.dormir(10 * time.Minute)
	return g, f
}

func TestTempoLimiteVerificarRclone(t *testing.T) {
	_, f := preparaDorminhoco(t)
	inicio := time.Now()

	err := verificarRclone(f.exe)

	dentroDoLimite(t, inicio)
	eTempoEsgotado(t, err)
	if !strings.Contains(err.Error(), "rclone --version") {
		t.Errorf("a mensagem deveria citar o comando: %v", err)
	}
	semProcessoVivo(t, f)
}

func TestTempoLimiteTestarRclone(t *testing.T) {
	_, f := preparaDorminhoco(t)
	inicio := time.Now()

	if testarRclone(f.exe) {
		t.Error("rclone travado não pode contar como disponível")
	}

	dentroDoLimite(t, inicio)
	semProcessoVivo(t, f)
}

func TestTempoLimiteObscurecerSenha(t *testing.T) {
	g, f := preparaDorminhoco(t)
	inicio := time.Now()

	_, err := g.ObscurecerSenha("segredo")

	dentroDoLimite(t, inicio)
	eTempoEsgotado(t, err)
	if strings.Contains(err.Error(), "segredo") {
		t.Errorf("a mensagem não pode levar a senha: %v", err)
	}
	semProcessoVivo(t, f)
}

func TestTempoLimiteCriarRemoto(t *testing.T) {
	g, f := preparaDorminhoco(t)
	inicio := time.Now()

	ok, msg := g.CriarRemoto("meudrive", "drive", map[string]string{"token": "token-secreto"})

	dentroDoLimite(t, inicio)
	if ok || !strings.Contains(msg, "tempo esgotado") || !strings.Contains(msg, "rclone config create") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
	if strings.Contains(msg, "token-secreto") {
		t.Errorf("a mensagem não pode levar o token: %q", msg)
	}
	semProcessoVivo(t, f)
}

func TestTempoLimiteCriarCrypt(t *testing.T) {
	g, f := preparaDorminhoco(t)
	inicio := time.Now()

	ok, msg := g.CriarCrypt("cofre", "b:", "segredo", "", nil)

	// O primeiro passo (obscure) já esgota; nada mais é chamado.
	dentroDoLimite(t, inicio)
	if ok || !strings.Contains(msg, "tempo esgotado") || strings.Contains(msg, "segredo") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
	semProcessoVivo(t, f)
}

func TestTempoLimiteRemoverRemoto(t *testing.T) {
	g, f := preparaDorminhoco(t)
	inicio := time.Now()

	ok, msg := g.RemoverRemoto("cofre")

	dentroDoLimite(t, inicio)
	if ok || !strings.Contains(msg, "tempo esgotado") || !strings.Contains(msg, "rclone config delete") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
	semProcessoVivo(t, f)
}

// As listagens ainda não devolvem erro (isso é a demanda 009). Aqui se confere
// que elas voltam dentro do limite, sem processo vivo, e que a função interna
// que elas usam devolve o erro de tempo esgotado.

func TestTempoLimiteConfigDump(t *testing.T) {
	g, f := preparaDorminhoco(t)
	inicio := time.Now()

	_, err := g.configDump()

	dentroDoLimite(t, inicio)
	eTempoEsgotado(t, err)
	semProcessoVivo(t, f)
}

func TestTempoLimiteListarRemotos(t *testing.T) {
	for nome, chamar := range map[string]func(g *GerenciadorRClone) bool{
		"ListarRemotos":          func(g *GerenciadorRClone) bool { return g.ListarRemotos() == nil },
		"ListarRemotosDetalhado": func(g *GerenciadorRClone) bool { return g.ListarRemotosDetalhado() == nil },
		"ObterConfigRemoto":      func(g *GerenciadorRClone) bool { return g.ObterConfigRemoto("cofre") == nil },
		"ListarTodosRemotos":     func(g *GerenciadorRClone) bool { return g.ListarTodosRemotos() == nil },
		"ListarDiretoriosRemoto": func(g *GerenciadorRClone) bool { return g.ListarDiretoriosRemoto("gdrive", "") == nil },
	} {
		t.Run(nome, func(t *testing.T) {
			g, f := preparaDorminhoco(t)
			inicio := time.Now()

			vazio := chamar(g)

			dentroDoLimite(t, inicio)
			if !vazio {
				t.Error("esperava nil depois do tempo esgotado")
			}
			semProcessoVivo(t, f)
		})
	}
}

func TestTempoLimiteListarTodosRemotosInterno(t *testing.T) {
	g, f := preparaDorminhoco(t)
	inicio := time.Now()

	_, err := g.listarTodosRemotos()

	dentroDoLimite(t, inicio)
	eTempoEsgotado(t, err)
	semProcessoVivo(t, f)
}

func TestTempoLimiteListagemDePastas(t *testing.T) {
	g, f := preparaDorminhoco(t)
	inicio := time.Now()

	_, err := chamadaRclone{executavel: g.Executavel, args: []string{"lsd", "gdrive:"}, limite: limiteListagem}.rodar()

	dentroDoLimite(t, inicio)
	eTempoEsgotado(t, err)
	if !strings.Contains(err.Error(), "rclone lsd") || strings.Contains(err.Error(), "gdrive") {
		t.Errorf("a mensagem deveria citar só o subcomando: %v", err)
	}
	semProcessoVivo(t, f)
}

func TestTempoLimiteOAuthIniciar(t *testing.T) {
	_, f := preparaDorminhoco(t)
	g := NovoGerenciadorOAuth()
	inicio := time.Now()

	if !g.Iniciar(f.exe, "drive") {
		t.Fatal("Iniciar devolveu false")
	}
	if !esperarAte(limiteTeste+time.Second, func() bool { return g.ObterStatus().Erro != "" }) {
		t.Fatalf("authorize não esgotou o tempo: %+v", g.ObterStatus())
	}

	dentroDoLimite(t, inicio)
	st := g.ObterStatus()
	if !strings.Contains(st.Erro, "tempo esgotado") || !strings.Contains(st.Erro, "rclone authorize") || st.Concluido {
		t.Errorf("status = %+v", st)
	}
	semProcessoVivo(t, f)
}

func TestDescreverComandoNaoLevaValores(t *testing.T) {
	casos := map[string][]string{
		"config create": {"config", "create", "x", "drive", "token", "{segredo}"},
		"obscure":       {"obscure", "-"},
		"lsd":           {"lsd", "remoto:pasta"},
		"":              nil,
	}
	for quer, args := range casos {
		if got := descreverComando(args); got != quer {
			t.Errorf("descreverComando(%v) = %q, quer %q", args, got, quer)
		}
	}
}

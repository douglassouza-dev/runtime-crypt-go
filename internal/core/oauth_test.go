package core

import (
	"runtime"
	"testing"
	"time"
)

// Testes de caracterização de oauth.go (demanda 016).

func TestNovoGerenciadorOAuthComecaVazio(t *testing.T) {
	st := NovoGerenciadorOAuth().ObterStatus()
	if st.URL != "" || st.Token != "" || st.Concluido {
		t.Errorf("status = %+v", st)
	}
}

func TestIniciarLeURLEToken(t *testing.T) {
	f := novoRcloneFalso(t)
	t.Setenv(envFalsoToken, `{"access_token":"abc","expiry":"2030-01-01"}`)
	g := NovoGerenciadorOAuth()

	if !g.Iniciar(f.exe, "drive") {
		t.Fatal("Iniciar devolveu false")
	}

	if !esperarAte(5*time.Second, func() bool { return g.ObterStatus().Concluido }) {
		t.Fatalf("token não chegou: %+v", g.ObterStatus())
	}
	st := g.ObterStatus()
	if st.URL != "http://127.0.0.1:53682/auth?state=falso" {
		t.Errorf("URL = %q", st.URL)
	}
	if st.Token != `{"access_token":"abc","expiry":"2030-01-01"}` {
		t.Errorf("Token = %q", st.Token)
	}
	if cs := f.chamadas(); len(cs) != 1 || !argsContem(cs[0].Args, "authorize", "drive") {
		t.Errorf("chamadas = %+v", cs)
	}
}

func TestIniciarExecutavelInexistente(t *testing.T) {
	g := NovoGerenciadorOAuth()
	if g.Iniciar(t.TempDir()+"/nao-existe", "drive") {
		t.Error("Iniciar deveria devolver false")
	}
}

func TestObterStatusSoComURL(t *testing.T) {
	f := novoRcloneFalso(t)
	g := NovoGerenciadorOAuth()
	if !g.Iniciar(f.exe, "drive") {
		t.Fatal("Iniciar devolveu false")
	}
	// Sem Abortar aqui: ver TestAbortarEncerraOProcesso. O Cleanup do falso
	// mata o processo e a goroutine de Iniciar faz o Wait.

	if !esperarAte(5*time.Second, func() bool { return g.ObterStatus().URL != "" }) {
		t.Fatal("URL não chegou")
	}
	if st := g.ObterStatus(); st.Concluido || st.Token != "" {
		t.Errorf("sem token ainda: %+v", st)
	}
}

func TestAbortarSemProcessoNaoFazNada(t *testing.T) {
	g := NovoGerenciadorOAuth()
	g.Abortar()
	if st := g.ObterStatus(); st.Concluido {
		t.Errorf("status = %+v", st)
	}
}

func TestAbortarEncerraOProcesso(t *testing.T) {
	f := novoRcloneFalso(t)
	g := NovoGerenciadorOAuth()
	if !g.Iniciar(f.exe, "drive") {
		t.Fatal("Iniciar devolveu false")
	}
	cs := f.esperarChamadas(1, 5*time.Second)
	if len(cs) != 1 {
		t.Fatalf("chamadas = %+v", cs)
	}

	g.Abortar()

	if !esperarAte(2*time.Second, func() bool { return !processoVivoNoSO(cs[0].Pid) }) {
		t.Errorf("authorize %d continua vivo depois de Abortar", cs[0].Pid)
	}
}

// fakeNoPath põe o falso no PATH com os nomes que AbrirNavegador e
// AbrirExplorador procuram.
func fakeNoPath(t *testing.T) *rcloneFalso {
	f := novoRcloneFalso(t)
	dir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		f.copiarComo(dir, "cmd")
		f.copiarComo(dir, "explorer")
	case "darwin":
		f.copiarComo(dir, "open")
	default:
		f.copiarComo(dir, "xdg-open")
	}
	t.Setenv("PATH", dir)
	return f
}

func TestAbrirNavegador(t *testing.T) {
	f := fakeNoPath(t)

	if err := AbrirNavegador("http://127.0.0.1:1/x"); err != nil {
		t.Fatal(err)
	}

	cs := f.esperarChamadas(1, 5*time.Second)
	if len(cs) != 1 || !argsContem(cs[0].Args, "http://127.0.0.1:1/x") {
		t.Errorf("chamadas = %+v", cs)
	}
	if runtime.GOOS == "windows" && (cs[0].Programa != "cmd" || !argsContem(cs[0].Args, "/c", "start", "")) {
		t.Errorf("no Windows deveria ser `cmd /c start \"\" url`: %+v", cs[0])
	}
}

func TestAbrirExplorador(t *testing.T) {
	f := fakeNoPath(t)
	alvo := t.TempDir()

	if err := AbrirExplorador(alvo); err != nil {
		t.Fatal(err)
	}

	cs := f.esperarChamadas(1, 5*time.Second)
	if len(cs) != 1 || !argsContem(cs[0].Args, alvo) {
		t.Errorf("chamadas = %+v", cs)
	}
}

func TestIniciarDuasVezesEncerraOPrimeiro(t *testing.T) {
	f := novoRcloneFalso(t)
	g := NovoGerenciadorOAuth()

	if !g.Iniciar(f.exe, "drive") {
		t.Fatal("primeiro Iniciar devolveu false")
	}
	primeiro := f.esperarChamadas(1, 5*time.Second)
	if len(primeiro) != 1 {
		t.Fatalf("chamadas = %+v", primeiro)
	}
	if !g.Iniciar(f.exe, "onedrive") {
		t.Fatal("segundo Iniciar devolveu false")
	}
	cs := f.esperarChamadas(2, 5*time.Second)
	if len(cs) != 2 {
		t.Fatalf("chamadas = %+v", cs)
	}

	if processoVivoNoSO(cs[0].Pid) {
		t.Errorf("o primeiro authorize %d continua vivo depois do segundo Iniciar", cs[0].Pid)
	}
	if !processoVivoNoSO(cs[1].Pid) {
		t.Errorf("o segundo authorize %d deveria estar vivo até Abortar", cs[1].Pid)
	}

	g.Abortar()

	if processoVivoNoSO(cs[1].Pid) {
		t.Errorf("o segundo authorize %d continua vivo depois de Abortar", cs[1].Pid)
	}
}

package core

import (
	"strings"
	"testing"
	"time"
)

// Demanda 013: mensagens com a unidade (Windows) ou com a pasta (Linux/macOS).

func comHome(t *testing.T, home string) {
	t.Helper()
	antes := diretorioHome
	diretorioHome = func() string { return home }
	t.Cleanup(func() { diretorioHome = antes })
}

func TestTextoERotuloDoPonto(t *testing.T) {
	comHome(t, "/home/ana")
	casos := []struct{ ponto, texto, rotulo string }{
		{`V:\`, `V:\`, `a unidade V:\`},
		{"V:", "V:", "a unidade V:"},
		{"/home/ana/RuntimeCrypto/fotos", "~/RuntimeCrypto/fotos", "a pasta ~/RuntimeCrypto/fotos"},
		{"/home/anabela/x", "/home/anabela/x", "a pasta /home/anabela/x"},
		{"/mnt/cofre", "/mnt/cofre", "a pasta /mnt/cofre"},
	}
	for _, c := range casos {
		if got := TextoPonto(c.ponto); got != c.texto {
			t.Errorf("TextoPonto(%q) = %q, quer %q", c.ponto, got, c.texto)
		}
		if got := RotuloPonto(c.ponto); got != c.rotulo {
			t.Errorf("RotuloPonto(%q) = %q, quer %q", c.ponto, got, c.rotulo)
		}
	}
}

// As mensagens de montar/desmontar usam o rótulo do ponto: com a pasta,
// nunca "Unidade /caminho".
func TestMensagensDeMontagemComPastaENaoUnidade(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	g.esperaEncerrar = 200 * time.Millisecond
	ponto, chave := "/home/ana/RuntimeCrypto/fotos", "/home/ana/RuntimeCrypto/fotos"
	g.caminhoPonto = func(l string) string { return ponto }
	if osWindows() {
		ponto, chave = `V:\`, "V"
	}
	comHome(t, "/home/ana")
	quer := RotuloPonto(ponto)

	ok, msg, _ := g.MontarUnidade("cofre", chave, "", nil)
	if !ok || msg != maiuscula(quer)+" foi montada." {
		t.Fatalf("montar: ok=%v msg=%q", ok, msg)
	}
	ok, msg = g.DesmontarUnidade(chave)
	if !ok || msg != maiuscula(quer)+" foi desmontada." {
		t.Errorf("desmontar: ok=%v msg=%q", ok, msg)
	}
	if !osWindows() && (strings.Contains(msg, "Unidade") || !strings.Contains(msg, "~/RuntimeCrypto/fotos")) {
		t.Errorf("mensagem de pasta: %q", msg)
	}
}

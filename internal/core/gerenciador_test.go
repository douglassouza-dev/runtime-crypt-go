package core

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Testes de caracterização de gerenciador.go (demanda 016). Registram o
// comportamento de hoje. Onde hoje há defeito, o teste certo existe e fica
// com t.Skip citando a demanda que corrige.

func novoGerenciadorFalso(t *testing.T) (*GerenciadorRClone, *rcloneFalso) {
	t.Helper()
	f := novoRcloneFalso(t)
	g := NovoGerenciadorEm(t.TempDir(), f.exe)
	g.Montagens.pontoExiste = pontoPeloFalso(f)
	return g, f
}

func TestNovoGerenciadorEmUsaExecutavelEDiretorioInformados(t *testing.T) {
	f := novoRcloneFalso(t)
	dir := t.TempDir()

	g := NovoGerenciadorEm(dir, f.exe)

	if g.Executavel != f.exe {
		t.Errorf("Executavel = %q, quer %q", g.Executavel, f.exe)
	}
	if g.DiretorioApp != dir {
		t.Errorf("DiretorioApp = %q, quer %q", g.DiretorioApp, dir)
	}
	if g.Cofres == nil || g.Montagens == nil || g.Senhas == nil || g.OAuth == nil || g.Vfs == nil {
		t.Fatal("algum subgerenciador ficou nil")
	}
	if n := len(f.chamadas()); n != 0 {
		t.Errorf("com executável informado não deveria testar o rclone; houve %d chamadas", n)
	}
}

func TestNovoGerenciadorEmSemExecutavelProcuraNoPath(t *testing.T) {
	f := novoRcloneFalso(t)
	pathDir := t.TempDir()
	f.copiarComo(pathDir, "rclone")
	t.Setenv("PATH", pathDir)

	g := NovoGerenciadorEm(t.TempDir(), "")

	quer := "rclone"
	if runtime.GOOS == "windows" {
		quer = "rclone.exe"
	}
	if g.Executavel != quer {
		t.Fatalf("Executavel = %q, quer %q (achado pelo PATH)", g.Executavel, quer)
	}
	cs := f.chamadas()
	if len(cs) == 0 || !argsContem(cs[len(cs)-1].Args, "--version") {
		t.Errorf("esperava uma chamada `rclone --version`, houve %+v", cs)
	}
}

func TestNovoGerenciadorSemRcloneFicaIndisponivel(t *testing.T) {
	novoRcloneFalso(t)
	t.Setenv("PATH", t.TempDir())

	g := NovoGerenciador()

	if g.EstaDisponivel() {
		t.Fatalf("sem rclone no diretório do app nem no PATH, EstaDisponivel deveria ser false (Executavel=%q)", g.Executavel)
	}
	if g.DiretorioApp == "" {
		t.Error("DiretorioApp vazio")
	}
}

func TestEstaDisponivel(t *testing.T) {
	g := &GerenciadorRClone{}
	if g.EstaDisponivel() {
		t.Error("Executavel vazio deveria ser indisponível")
	}
	g.Executavel = "qualquer"
	if !g.EstaDisponivel() {
		t.Error("Executavel preenchido deveria ser disponível")
	}
}

func TestSemRcloneTodasAsFuncoesRecusam(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // nem procura o rclone real da máquina
	g := NovoGerenciadorEm(t.TempDir(), "")
	if g.EstaDisponivel() {
		t.Fatalf("Executavel = %q, esperava vazio", g.Executavel)
	}

	if _, err := g.ObscurecerSenha("x"); err == nil {
		t.Error("ObscurecerSenha deveria falhar")
	}
	if ok, _ := g.CriarRemoto("a", "drive", nil); ok {
		t.Error("CriarRemoto deveria falhar")
	}
	if ok, _ := g.CriarCrypt("a", "b:", "s", "", nil); ok {
		t.Error("CriarCrypt deveria falhar")
	}
	if ok, _ := g.ImportarCrypt("a", "b:", "s", "", nil); ok {
		t.Error("ImportarCrypt deveria falhar")
	}
	if ok, _ := g.RemoverRemoto("a"); ok {
		t.Error("RemoverRemoto deveria falhar")
	}
	erros := map[string]error{}
	_, erros["ListarRemotos"] = g.ListarRemotos()
	_, erros["ListarTodosRemotos"] = g.ListarTodosRemotos()
	_, erros["ListarRemotosDetalhado"] = g.ListarRemotosDetalhado()
	_, erros["ObterConfigRemoto"] = g.ObterConfigRemoto("a")
	_, erros["ListarDiretoriosRemoto"] = g.ListarDiretoriosRemoto("a", "")
	for nome, err := range erros {
		if !errors.Is(err, ErrRcloneIndisponivel) {
			t.Errorf("%s sem rclone: err = %v, quer ErrRcloneIndisponivel", nome, err)
		}
	}
}

func TestObscurecerSenhaPassaPeloStdin(t *testing.T) {
	g, f := novoGerenciadorFalso(t)

	obs, err := g.ObscurecerSenha("segredo")

	if err != nil {
		t.Fatal(err)
	}
	if obs != "obs(segredo)" {
		t.Errorf("obs = %q", obs)
	}
	cs := f.chamadas()
	if len(cs) != 1 || !argsContem(cs[0].Args, "obscure", "-") || cs[0].Stdin != "segredo" {
		t.Errorf("chamada inesperada: %+v", cs)
	}
	for _, a := range cs[0].Args {
		if strings.Contains(a, "segredo") {
			t.Errorf("senha em texto puro apareceu nos argumentos: %v", cs[0].Args)
		}
	}
}

func TestObscurecerSenhaErroDoRclone(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.falhar()
	if _, err := g.ObscurecerSenha("x"); err == nil {
		t.Error("esperava erro")
	}
}

func TestCriarRemoto(t *testing.T) {
	g, f := novoGerenciadorFalso(t)

	ok, msg := g.CriarRemoto("meudrive", "drive", map[string]string{"token": "{}", "vazio": ""})

	if !ok || !strings.Contains(msg, "meudrive") {
		t.Fatalf("ok=%v msg=%q", ok, msg)
	}
	args := f.chamadas()[0].Args
	if !argsContem(args, "config", "create", "meudrive", "drive") || !argsContem(args, "token", "{}") {
		t.Errorf("args = %v", args)
	}
	for _, a := range args {
		if a == "vazio" {
			t.Errorf("parâmetro com valor vazio não deveria ir para o rclone: %v", args)
		}
	}
}

func TestCriarRemotoDevolveStderrDoRclone(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.falhar()

	ok, msg := g.CriarRemoto("x", "drive", nil)

	if ok || !strings.Contains(msg, "CRITICAL: falha do rclone falso") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
}

func TestCriarCrypt(t *testing.T) {
	g, f := novoGerenciadorFalso(t)

	ok, msg := g.CriarCrypt("cofre", "gdrive:pasta", "s1", "", nil)

	if !ok {
		t.Fatalf("msg=%q", msg)
	}
	cs := f.chamadas()
	if len(cs) != 2 {
		t.Fatalf("esperava 1 obscure + 1 config create, houve %d: %+v", len(cs), cs)
	}
	args := cs[1].Args
	for _, par := range [][]string{
		{"config", "create", "cofre", "crypt"},
		{"remote", "gdrive:pasta"},
		{"password", "obs(s1)"},
		{"password2", "obs(s1)"}, // sem senha2, repete a primeira
		{"filename_encryption", "standard"},
		{"directory_name_encryption", "true"},
	} {
		if !argsContem(args, par...) {
			t.Errorf("faltou %v em %v", par, args)
		}
	}
	if argsContem(args, "no_data_encryption") {
		t.Errorf("no_data_encryption só vai quando true: %v", args)
	}
}

func TestCriarCryptComSenha2EConfig(t *testing.T) {
	g, f := novoGerenciadorFalso(t)

	ok, _ := g.CriarCrypt("cofre", "b:", "s1", "s2", map[string]string{"filename_encryption": "off", "no_data_encryption": "true"})

	if !ok {
		t.Fatal("esperava sucesso")
	}
	cs := f.chamadas()
	args := cs[len(cs)-1].Args
	for _, par := range [][]string{{"password2", "obs(s2)"}, {"filename_encryption", "off"}, {"no_data_encryption", "true"}} {
		if !argsContem(args, par...) {
			t.Errorf("faltou %v em %v", par, args)
		}
	}
}

func TestCriarCryptNaoPoeSenhaNosArgumentos(t *testing.T) {
	t.Skip("defeito conhecido: senha ofuscada vai nos argumentos do processo — demanda 005")
	g, f := novoGerenciadorFalso(t)
	g.CriarCrypt("cofre", "b:", "s1", "", nil)
	for _, c := range f.chamadas() {
		for _, a := range c.Args {
			if strings.Contains(a, "obs(") {
				t.Fatalf("senha ofuscada nos argumentos: %v", c.Args)
			}
		}
	}
}

func TestImportarCryptFazOMesmoQueCriarCrypt(t *testing.T) {
	g, f := novoGerenciadorFalso(t)

	ok, _ := g.ImportarCrypt("antigo", "b:x", "s1", "", nil)

	if !ok {
		t.Fatal("esperava sucesso")
	}
	cs := f.chamadas()
	if !argsContem(cs[len(cs)-1].Args, "config", "create", "antigo", "crypt") {
		t.Errorf("args = %v", cs[len(cs)-1].Args)
	}
}

func TestRemoverRemoto(t *testing.T) {
	g, f := novoGerenciadorFalso(t)

	ok, msg := g.RemoverRemoto("cofre:")

	if !ok || !strings.Contains(msg, "cofre") {
		t.Fatalf("ok=%v msg=%q", ok, msg)
	}
	if args := f.chamadas()[0].Args; !argsContem(args, "config", "delete", "cofre") || len(args) != 3 {
		t.Errorf("args = %v (o ':' final deveria sair)", args)
	}

	f.falhar()
	if ok, msg := g.RemoverRemoto("cofre"); ok || !strings.Contains(msg, "CRITICAL") {
		t.Errorf("falha: ok=%v msg=%q", ok, msg)
	}
}

const dumpExemplo = `{
  "gdrive": {"type": "drive", "token": "{}"},
  "cofre": {"type": "crypt", "remote": "gdrive:cofre", "password": "x"},
  "outro": {"type": "crypt", "remote": "gdrive:outro"}
}`

func TestListarRemotosSoCrypt(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("dump.json", dumpExemplo)

	got, err := g.ListarRemotos()

	if err != nil || len(got) != 2 || !contem(got, "cofre:") || !contem(got, "outro:") {
		t.Errorf("ListarRemotos = %v", got)
	}
}

// Demanda 009: erro do rclone volta como erro, com o motivo que ele escreveu.
func TestListagensDeRemotosDevolvemErroDoRclone(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.falhar()

	erros := map[string]error{}
	_, erros["ListarRemotos"] = g.ListarRemotos()
	_, erros["ListarTodosRemotos"] = g.ListarTodosRemotos()
	_, erros["ListarRemotosDetalhado"] = g.ListarRemotosDetalhado()
	_, erros["ObterConfigRemoto"] = g.ObterConfigRemoto("cofre")
	for nome, err := range erros {
		if err == nil || !strings.Contains(err.Error(), "falha do rclone falso") {
			t.Errorf("%s: err = %v, quer o motivo do rclone", nome, err)
		}
	}
}

func TestListarTodosRemotos(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("dump.json", dumpExemplo)

	got, err := g.ListarTodosRemotos()

	if err != nil || strings.Join(got, ",") != "cofre:,gdrive:,outro:" {
		t.Errorf("ListarTodosRemotos = %v", got)
	}
}

func TestListarRemotosDetalhado(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("dump.json", dumpExemplo)

	got, err := g.ListarRemotosDetalhado()

	if err != nil || len(got) != 3 {
		t.Fatalf("len = %d: %+v", len(got), got)
	}
	for _, r := range got {
		if r.Nome == "cofre" && (!r.IsCrypt || r.RemotoBase != "gdrive:cofre" || r.Tipo != "crypt" || r.Estado != EstadoDesmontado) {
			t.Errorf("cofre: %+v", r)
		}
		if r.Nome == "gdrive" && (r.IsCrypt || r.Tipo != "drive") {
			t.Errorf("gdrive: %+v", r)
		}
	}
}

func TestObterConfigRemoto(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("dump.json", dumpExemplo)

	cfg, err := g.ObterConfigRemoto("cofre:")

	if err != nil || cfg["remote"] != "gdrive:cofre" {
		t.Errorf("cfg = %v, err = %v", cfg, err)
	}
	if cfg, err := g.ObterConfigRemoto("naoexiste"); cfg != nil || !errors.Is(err, ErrRemotoNaoEncontrado) {
		t.Errorf("remoto inexistente: cfg = %v, err = %v", cfg, err)
	}
}

// lsjsonReal é a saída de `rclone lsjson --dirs-only` do rclone v1.60.1 para
// uma pasta com estas subpastas e um arquivo solto (que não aparece).
const lsjsonReal = `[
{"Path":"  dois  espaços","Name":"  dois  espaços","Size":-1,"MimeType":"inode/directory","ModTime":"2026-10-09T18:56:36.838441820-03:00","IsDir":true},
{"Path":"-rascunho","Name":"-rascunho","Size":-1,"MimeType":"inode/directory","ModTime":"2026-10-09T18:56:36.838441820-03:00","IsDir":true},
{"Path":"2024 fotos","Name":"2024 fotos","Size":-1,"MimeType":"inode/directory","ModTime":"2026-10-09T18:56:36.838441820-03:00","IsDir":true},
{"Path":"alfa","Name":"alfa","Size":-1,"MimeType":"inode/directory","ModTime":"2026-10-09T18:56:36.838441820-03:00","IsDir":true},
{"Path":"ação","Name":"ação","Size":-1,"MimeType":"inode/directory","ModTime":"2026-10-09T18:56:36.838441820-03:00","IsDir":true},
{"Path":"minha pasta","Name":"minha pasta","Size":-1,"MimeType":"inode/directory","ModTime":"2026-10-09T18:56:36.838441820-03:00","IsDir":true}
]
`

func TestListarDiretoriosRemoto(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("lsjson.json", `[{"Path":"zeta","Name":"zeta","Size":-1,"IsDir":true},{"Path":"alfa","Name":"alfa","Size":-1,"IsDir":true}]`)

	got, err := g.ListarDiretoriosRemoto("gdrive", "/fotos")

	if err != nil || strings.Join(got, ",") != "alfa,zeta" {
		t.Errorf("dirs = %v", got)
	}
	if args := f.chamadas()[0].Args; !argsContem(args, "lsjson", "--dirs-only", "gdrive:fotos") {
		t.Errorf("args = %v", args)
	}
}

// Formato real: a saída que o rclone de verdade imprime (demanda 021). Antes
// da 021 o core usava `lsd` e recortava as colunas alinhadas, devolvendo
// "-1 alfa".
func TestListarDiretoriosRemotoFormatoRealDoRclone(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("lsjson.json", lsjsonReal)

	got, err := g.ListarDiretoriosRemoto("gdrive", "")
	if err != nil {
		t.Fatal(err)
	}

	quer := []string{"  dois  espaços", "-rascunho", "2024 fotos", "alfa", "ação", "minha pasta"}
	if len(got) != len(quer) {
		t.Fatalf("dirs = %q, quer %q", got, quer)
	}
	for i := range quer {
		if got[i] != quer[i] {
			t.Errorf("dirs[%d] = %q, quer %q (nome exato, sem tirar nem pôr caractere)", i, got[i], quer[i])
		}
	}
}

func TestListarDiretoriosRemotoIgnoraArquivos(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("lsjson.json", `[{"Name":"pasta","IsDir":true},{"Name":"arquivo.txt","IsDir":false}]`)

	if got, err := g.ListarDiretoriosRemoto("gdrive", ""); err != nil || strings.Join(got, "|") != "pasta" {
		t.Errorf("dirs = %q, err = %v", got, err)
	}
}

func TestListarDiretoriosRemotoPastaVazia(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("lsjson.json", "[]\n")

	got, err := g.ListarDiretoriosRemoto("gdrive", "")
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("pasta vazia: dirs = %q, err = %v; quer lista vazia sem erro", got, err)
	}
}

// Demanda 009: executável falso que sai com código 1 devolve erro, com a
// última linha do stderr como motivo e sem a data do log.
func TestListarDiretoriosRemotoErroDoRcloneDevolveErro(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.falhar()
	t.Setenv(envFalsoStderr, "2026/10/09 23:10:00 CRITICAL: Failed to lsjson: couldn't list directory: invalid_grant")

	got, err := g.ListarDiretoriosRemoto("gdrive:", "")

	if err == nil {
		t.Fatalf("esperava erro, veio dirs = %q", got)
	}
	if got != nil {
		t.Errorf("com erro, dirs deveria ser nil; veio %q", got)
	}
	if quer := "CRITICAL: Failed to lsjson: couldn't list directory: invalid_grant"; err.Error() != quer {
		t.Errorf("motivo = %q, quer %q", err.Error(), quer)
	}
	var saida *exec.ExitError
	if !errors.As(err, &saida) {
		t.Errorf("o erro deveria embrulhar o *exec.ExitError: %T", err)
	}
}

func TestListarDiretoriosRemotoSaidaIlegivelDevolveErro(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.escrever("lsjson.json", "isto não é json")

	if got, err := g.ListarDiretoriosRemoto("gdrive", ""); err == nil {
		t.Errorf("saída ilegível deveria dar erro; veio %q", got)
	}
}

// Mesmo critério de `rg -n "return nil$" internal/core/gerenciador.go`, sem
// depender do rg instalado.
func TestGerenciadorSemRetornoNilDisfarcado(t *testing.T) {
	dados, err := os.ReadFile("gerenciador.go")
	if err != nil {
		t.Fatal(err)
	}
	for i, linha := range strings.Split(strings.ReplaceAll(string(dados), "\r\n", "\n"), "\n") {
		if strings.HasSuffix(linha, "return nil") {
			t.Errorf("gerenciador.go:%d: %q", i+1, strings.TrimSpace(linha))
		}
	}
}

func TestListarCofres(t *testing.T) {
	f := novoRcloneFalso(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ArquivoCofres), []byte(`[{"nome":"cofre","provedor_id":"drive"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	g := NovoGerenciadorEm(dir, f.exe)
	g.Senhas.Armazenar("cofre", "s")

	got := g.ListarCofres()

	if len(got) != 1 || got[0].Nome != "cofre" || got[0].Estado != EstadoDesmontado || !got[0].TemSenha {
		t.Errorf("ListarCofres = %+v", got)
	}
}

func TestEncerrarDesmontaELimpaSenhas(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	g.Senhas.Armazenar("cofre", "s")
	if ok, msg, _ := g.Montagens.MontarUnidade("cofre", "V", "s", nil); !ok {
		t.Fatal(msg)
	}
	pid := pidDaMontagem(t, g.Montagens, "V")

	g.Encerrar()

	if processoVivoNoSO(pid) {
		t.Errorf("Encerrar deixou o processo de montagem %d vivo", pid)
	}
	if g.Senhas.Existe("cofre") {
		t.Error("Encerrar deveria limpar as senhas")
	}
}

func TestEncerrarDepoisDaTelaAtualizarDesmonta(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	if ok, msg, _ := g.Montagens.MontarUnidade("cofre", "V", "s", nil); !ok {
		t.Fatal(msg)
	}
	pid := pidDaMontagem(t, g.Montagens, "V")

	g.ListarCofres() // o que a tela faz a cada 3 s
	g.Encerrar()

	if processoVivoNoSO(pid) {
		t.Errorf("Encerrar deixou o processo de montagem %d vivo", pid)
	}
}

func contem(lista []string, s string) bool {
	for _, x := range lista {
		if x == s {
			return true
		}
	}
	return false
}

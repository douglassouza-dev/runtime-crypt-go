package core

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

// Testes da demanda 031.

const cofresAntigos = `[{"nome":"antigo","provedor_id":"drive","provedor_nome":"Google Drive","remoto_base":"antigo_base:"}]`
const cofresNovos = `[{"nome":"novo","provedor_id":"drive","provedor_nome":"Google Drive","remoto_base":"novo_base:"}]`

var dataAntiga = time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)

// basePastaConfig troca os.UserConfigDir por uma pasta temporária e devolve
// a pasta de configuração esperada.
func basePastaConfig(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	antes := pastaConfigBase
	pastaConfigBase = func() (string, error) { return base, nil }
	t.Cleanup(func() { pastaConfigBase = antes })
	return filepath.Join(base, NomePastaConfig)
}

// gravarComData grava o arquivo e põe uma data antiga, para conferir depois
// que ninguém o regravou.
func gravarComData(t *testing.T, caminho, conteudo string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(caminho, dataAntiga, dataAntiga); err != nil {
		t.Fatal(err)
	}
}

// intacto confere conteúdo e data de modificação.
func intacto(t *testing.T, caminho, conteudo string) {
	t.Helper()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("%s: %v", caminho, err)
	}
	if string(dados) != conteudo {
		t.Errorf("%s mudou: %q", caminho, dados)
	}
	info, err := os.Stat(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(dataAntiga) {
		t.Errorf("%s foi regravado (mtime %v)", caminho, info.ModTime())
	}
}

func nomesNaPasta(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var ns []string
	for _, e := range es {
		ns = append(ns, e.Name())
	}
	sort.Strings(ns)
	return ns
}

func nomesCofres(g *GerenciadorRClone) []string {
	var ns []string
	for _, c := range g.Cofres.Listar(nil, g.Senhas) {
		ns = append(ns, c.Nome)
	}
	return ns
}

func TestCaminhoPastaConfig(t *testing.T) {
	quer := basePastaConfig(t)
	if got, err := CaminhoPastaConfig(); err != nil || got != quer {
		t.Errorf("(%q, %v), quer %q", got, err, quer)
	}
	pastaConfigBase = func() (string, error) { return "", errors.New("sem HOME") }
	if _, err := CaminhoPastaConfig(); err == nil {
		t.Error("deveria falhar sem a pasta do usuário")
	}
}

// Só o antigo: copiado uma vez para a pasta nova; o antigo fica igual.
func TestMigracaoCopiaOAntigoSemMexerNele(t *testing.T) {
	nova := basePastaConfig(t)
	antiga := t.TempDir()
	gravarComData(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
	gravarComData(t, filepath.Join(antiga, ArquivoVfs), `{"vfs_cache_mode":"writes"}`)

	g := novoGerenciadorPadrao(antiga, "rclone-que-nao-existe")

	if g.ErroPastaConfig != nil || g.ErroCofres != nil {
		t.Fatalf("erros: %v / %v", g.ErroPastaConfig, g.ErroCofres)
	}
	if g.DiretorioConfig != nova || g.DiretorioApp != antiga {
		t.Errorf("config=%q app=%q", g.DiretorioConfig, g.DiretorioApp)
	}
	if ns := nomesCofres(g); len(ns) != 1 || ns[0] != "antigo" {
		t.Errorf("cofres = %v", ns)
	}
	if g.Vfs.Obter()["vfs_cache_mode"] != "writes" {
		t.Errorf("vfs.json não veio: %v", g.Vfs.Obter())
	}
	intacto(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
	intacto(t, filepath.Join(antiga, ArquivoVfs), `{"vfs_cache_mode":"writes"}`)
	if ns := nomesNaPasta(t, nova); len(ns) != 2 || ns[0] != ArquivoCofres || ns[1] != ArquivoVfs {
		t.Errorf("pasta nova tem %v (sobrou temporário?)", ns)
	}
	if len(g.RegistroPastaConfig) != 2 {
		t.Errorf("registro = %q", g.RegistroPastaConfig)
	}

	// Gravar depois vai para a pasta nova; o antigo continua igual.
	if ok, msg := g.Cofres.Adicionar("outro", "drive", "Google Drive", "outro_base:"); !ok {
		t.Fatal(msg)
	}
	intacto(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)

	// Segunda abertura: não copia de novo e vale o novo.
	g2 := novoGerenciadorPadrao(antiga, "rclone-que-nao-existe")
	if ns := nomesCofres(g2); len(ns) != 2 || len(g2.RegistroPastaConfig) != 0 {
		t.Errorf("segunda abertura: cofres=%v registro=%q", ns, g2.RegistroPastaConfig)
	}
}

// Os dois existem: vale o novo, e nenhum é regravado.
func TestMigracaoComOsDoisValeONovo(t *testing.T) {
	nova := basePastaConfig(t)
	antiga := t.TempDir()
	gravarComData(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
	gravarComData(t, filepath.Join(nova, ArquivoCofres), cofresNovos)

	g := novoGerenciadorPadrao(antiga, "rclone-que-nao-existe")

	if ns := nomesCofres(g); len(ns) != 1 || ns[0] != "novo" {
		t.Errorf("cofres = %v, quer o novo", ns)
	}
	intacto(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
	intacto(t, filepath.Join(nova, ArquivoCofres), cofresNovos)
}

// Ler nunca grava: abrir com os arquivos na pasta nova não muda a data deles.
func TestLeituraNaoGrava(t *testing.T) {
	nova := basePastaConfig(t)
	gravarComData(t, filepath.Join(nova, ArquivoCofres), cofresNovos)
	gravarComData(t, filepath.Join(nova, ArquivoVfs), `{"vfs_cache_mode":"full"}`)

	g := novoGerenciadorPadrao(t.TempDir(), "rclone-que-nao-existe")
	_ = g.Cofres.Listar(nil, g.Senhas)
	_ = g.Vfs.Obter()

	intacto(t, filepath.Join(nova, ArquivoCofres), cofresNovos)
	intacto(t, filepath.Join(nova, ArquivoVfs), `{"vfs_cache_mode":"full"}`)
}

// Nada antigo: a pasta nova é criada vazia.
func TestMigracaoSemNadaAntigo(t *testing.T) {
	nova := basePastaConfig(t)
	g := novoGerenciadorPadrao(t.TempDir(), "rclone-que-nao-existe")
	if g.ErroPastaConfig != nil {
		t.Fatal(g.ErroPastaConfig)
	}
	if ns := nomesNaPasta(t, nova); len(ns) != 0 {
		t.Errorf("pasta nova tem %v", ns)
	}
}

// A pasta nova não pode ser criada: lê o antigo, avisa com a frase fixa e não
// grava nada em lugar nenhum.
func TestPastaNovaSemGravacaoNaoVoltaParaAPastaDoExecutavel(t *testing.T) {
	base := t.TempDir()
	arquivo := filepath.Join(base, "nao-e-pasta")
	if err := os.WriteFile(arquivo, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	antes := pastaConfigBase
	pastaConfigBase = func() (string, error) { return arquivo, nil }
	t.Cleanup(func() { pastaConfigBase = antes })
	antiga := t.TempDir()
	gravarComData(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
	gravarComData(t, filepath.Join(antiga, ArquivoVfs), `{"vfs_cache_mode":"writes"}`)

	g := novoGerenciadorPadrao(antiga, "rclone-que-nao-existe")

	var e *ErroPastaConfig
	if !errors.As(g.ErroPastaConfig, &e) {
		t.Fatalf("ErroPastaConfig = %v", g.ErroPastaConfig)
	}
	quer := "Não deu para usar a pasta de configuração (" + filepath.Join(arquivo, NomePastaConfig) + "): mudanças nos cofres e nas configurações não serão salvas."
	if g.ErroPastaConfig.Error() != quer {
		t.Errorf("frase = %q", g.ErroPastaConfig.Error())
	}
	if ns := nomesCofres(g); len(ns) != 1 || ns[0] != "antigo" {
		t.Errorf("deveria ler o antigo; cofres = %v", ns)
	}
	if ok, _ := g.Cofres.Adicionar("x", "drive", "Google Drive", "x_base:"); ok {
		t.Error("Adicionar deveria recusar")
	}
	if _, err := g.Vfs.Atualizar(map[string]string{"vfs_cache_mode": "full"}); !errors.As(err, &e) {
		t.Errorf("Atualizar VFS: %v", err)
	}
	if _, err := g.Vfs.Restaurar(); !errors.As(err, &e) {
		t.Errorf("Restaurar VFS: %v", err)
	}
	intacto(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
	intacto(t, filepath.Join(antiga, ArquivoVfs), `{"vfs_cache_mode":"writes"}`)
	if ns := nomesNaPasta(t, antiga); len(ns) != 2 {
		t.Errorf("a pasta do executável ganhou arquivos: %v", ns)
	}
}

// Com a pasta nova inutilizável e o vaults.json antigo corrompido, nem a cópia
// .corrompido é gravada na pasta do executável.
func TestPastaNovaSemGravacaoNaoGuardaCopiaCorrompida(t *testing.T) {
	base := t.TempDir()
	arquivo := filepath.Join(base, "nao-e-pasta")
	if err := os.WriteFile(arquivo, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	antes := pastaConfigBase
	pastaConfigBase = func() (string, error) { return arquivo, nil }
	t.Cleanup(func() { pastaConfigBase = antes })
	antiga := t.TempDir()
	gravarComData(t, filepath.Join(antiga, ArquivoCofres), "{quebrado")

	g := novoGerenciadorPadrao(antiga, "rclone-que-nao-existe")

	if g.ErroCofres == nil {
		t.Error("deveria acusar o arquivo corrompido")
	}
	if ns := nomesNaPasta(t, antiga); len(ns) != 1 {
		t.Errorf("a pasta do executável ganhou arquivos: %v", ns)
	}
}

// Sem saber a pasta do usuário: mesma regra, lendo da pasta do executável.
func TestSemPastaDoUsuario(t *testing.T) {
	antes := pastaConfigBase
	pastaConfigBase = func() (string, error) { return "", errors.New("sem HOME") }
	t.Cleanup(func() { pastaConfigBase = antes })
	antiga := t.TempDir()
	gravarComData(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)

	g := novoGerenciadorPadrao(antiga, "rclone-que-nao-existe")

	if g.ErroPastaConfig == nil || g.ErroPastaConfig.Error() != TextoPastaConfig("") {
		t.Errorf("ErroPastaConfig = %v", g.ErroPastaConfig)
	}
	if ns := nomesCofres(g); len(ns) != 1 {
		t.Errorf("cofres = %v", ns)
	}
	if ok, _ := g.Cofres.Adicionar("x", "drive", "Google Drive", "x_base:"); ok {
		t.Error("Adicionar deveria recusar")
	}
	intacto(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
}

// A cópia falha no rename: não sobra arquivo pela metade na pasta nova, o
// antigo fica igual e nada é gravado.
func TestMigracaoQueFalhaNaoDeixaMetade(t *testing.T) {
	nova := basePastaConfig(t)
	antiga := t.TempDir()
	gravarComData(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
	antes := renomearArquivo
	renomearArquivo = func(string, string) error { return errors.New("rename recusado") }
	t.Cleanup(func() { renomearArquivo = antes })

	g := novoGerenciadorPadrao(antiga, "rclone-que-nao-existe")

	if g.ErroPastaConfig == nil {
		t.Fatal("deveria avisar")
	}
	if ns := nomesNaPasta(t, nova); len(ns) != 0 {
		t.Errorf("sobrou na pasta nova: %v", ns)
	}
	if ns := nomesCofres(g); len(ns) != 1 || ns[0] != "antigo" {
		t.Errorf("cofres = %v", ns)
	}
	intacto(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
}

// Pasta nova que existe mas não aceita gravação (Unix; root grava em tudo).
func TestPastaNovaSomenteLeitura(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permissão de pasta não vale aqui")
	}
	nova := basePastaConfig(t)
	if err := os.MkdirAll(nova, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(nova, 0o700) })
	antiga := t.TempDir()
	gravarComData(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)

	g := novoGerenciadorPadrao(antiga, "rclone-que-nao-existe")

	if g.ErroPastaConfig == nil {
		t.Fatal("deveria avisar")
	}
	if ns := nomesCofres(g); len(ns) != 1 {
		t.Errorf("cofres = %v", ns)
	}
	intacto(t, filepath.Join(antiga, ArquivoCofres), cofresAntigos)
	if ns := nomesNaPasta(t, antiga); len(ns) != 1 {
		t.Errorf("a pasta do executável ganhou arquivos: %v", ns)
	}
}

// NovoGerenciadorEm (testes) continua usando uma pasta só, sem cópia.
func TestNovoGerenciadorEmUsaAPastaDada(t *testing.T) {
	basePastaConfig(t)
	dir := t.TempDir()
	g := NovoGerenciadorEm(dir, "rclone-que-nao-existe")
	if g.DiretorioApp != dir || g.DiretorioConfig != dir || g.ErroPastaConfig != nil {
		t.Errorf("%+v", g)
	}
}

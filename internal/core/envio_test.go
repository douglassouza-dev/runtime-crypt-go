package core

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Testes da demanda 025: Trancar espera a VFS enviar tudo antes de encerrar o
// rclone. O mount falso responde vfs/stats com o que o teste escreve.

func filaFalsa(f *rcloneFalso, emAndamento, naFila, comErro int) {
	f.escrever("vfs_stats.json", fmt.Sprintf(`{"diskCache":{"uploadsInProgress":%d,"uploadsQueued":%d,"erroredFiles":%d}}`, emAndamento, naFila, comErro))
}

func montarParaEnvio(t *testing.T) (*GerenciadorRClone, *rcloneFalso, int) {
	t.Helper()
	g, f := novoGerenciadorFalso(t)
	if ok, msg := g.Cofres.Adicionar("cofre", "drive", "Google Drive", "cofre_base:"); !ok {
		t.Fatal(msg)
	}
	g.Senhas.Armazenar("cofre", "segredo")
	if ok, msg, _ := g.Montagens.MontarUnidade("cofre", "V", "segredo", nil); !ok {
		t.Fatal(msg)
	}
	g.Montagens.intervaloEnvio = 20 * time.Millisecond
	t.Cleanup(g.Montagens.DesmontarTodas)
	return g, f, pidDaMontagem(t, g.Montagens, normalizarPonto("V"))
}

func cofreListado(t *testing.T, g *GerenciadorRClone) CofreStatus {
	t.Helper()
	for _, c := range g.ListarCofres() {
		if c.Nome == "cofre" {
			return c
		}
	}
	t.Fatal("cofre não listado")
	return CofreStatus{}
}

func TestTrancarEsperaOEnvioTerminar(t *testing.T) {
	g, f, pid := montarParaEnvio(t)
	filaFalsa(f, 1, 1, 0)

	feito := make(chan error, 1)
	go func() { feito <- g.Trancar("cofre") }()

	if !esperarAte(5*time.Second, func() bool { return cofreListado(t, g).Enviando == 2 }) {
		t.Fatalf("a lista deveria dizer 2 arquivos enviando: %+v", cofreListado(t, g))
	}
	time.Sleep(300 * time.Millisecond)
	select {
	case err := <-feito:
		t.Fatalf("Trancar voltou com envio pendente: %v", err)
	default:
	}
	if !processoVivoNoSO(pid) {
		t.Fatal("o rclone foi encerrado com envio pendente")
	}
	if c := cofreListado(t, g); c.Estado != EstadoMontado {
		t.Errorf("enquanto envia o cofre continua montado: %+v", c)
	}

	filaFalsa(f, 0, 1, 0)
	if !esperarAte(5*time.Second, func() bool { return cofreListado(t, g).Enviando == 1 }) {
		t.Errorf("a contagem deveria cair para 1: %+v", cofreListado(t, g))
	}
	filaFalsa(f, 0, 0, 0)

	select {
	case err := <-feito:
		if err != nil {
			t.Fatalf("Trancar: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Trancar não voltou depois de a fila zerar")
	}
	if processoVivoNoSO(pid) {
		t.Error("o rclone deveria ter terminado")
	}
	if c := cofreListado(t, g); c.Estado != EstadoDesmontado || c.Enviando != 0 {
		t.Errorf("depois de trancar: %+v", c)
	}
	if g.Senhas.Obter("cofre") != "" {
		t.Error("trancado, a senha sai da sessão")
	}
}

func TestTrancarComEnvioQueFalhouNaoTranca(t *testing.T) {
	g, f, pid := montarParaEnvio(t)
	filaFalsa(f, 0, 1, 1)

	err := g.Trancar("cofre")

	if err == nil || !strings.Contains(err.Error(), "o envio de 1 arquivo falhou") {
		t.Fatalf("erro = %v", err)
	}
	if !processoVivoNoSO(pid) {
		t.Error("com envio falho o rclone não pode ser encerrado")
	}
	if c := cofreListado(t, g); c.Estado != EstadoMontado || c.Enviando != 0 {
		t.Errorf("o cofre continua destrancado: %+v", c)
	}
	if g.Senhas.Obter("cofre") != "segredo" {
		t.Error("a senha da sessão fica")
	}
}

func TestTrancarComEnvioParadoDesisteSemEncerrar(t *testing.T) {
	g, f, pid := montarParaEnvio(t)
	g.Montagens.esperaSemEnvio = 300 * time.Millisecond
	filaFalsa(f, 1, 0, 0)

	err := g.Trancar("cofre")

	if err == nil || !strings.Contains(err.Error(), "parou") {
		t.Fatalf("erro = %v", err)
	}
	if !processoVivoNoSO(pid) || g.Senhas.Obter("cofre") != "segredo" {
		t.Error("envio parado: o cofre continua aberto e com a senha")
	}
}

// Um arquivo grande: a fila não muda, mas os bytes andam. Não desiste.
func TestTrancarComEnvioLentoQueAndaContinuaEsperando(t *testing.T) {
	g, f, _ := montarParaEnvio(t)
	g.Montagens.esperaSemEnvio = 300 * time.Millisecond
	filaFalsa(f, 1, 0, 0)
	var parar atomic.Bool
	go func() {
		for b := 1; !parar.Load(); b++ {
			f.escrever("core_stats.json", fmt.Sprintf(`{"bytes":%d}`, b*1000))
			time.Sleep(50 * time.Millisecond)
		}
	}()
	defer parar.Store(true)
	time.AfterFunc(1200*time.Millisecond, func() { filaFalsa(f, 0, 0, 0) })

	if err := g.Trancar("cofre"); err != nil {
		t.Fatalf("Trancar desistiu de um envio que andava: %v", err)
	}
}

// As credenciais do rc vão no ambiente, não nos argumentos (que aparecem na
// lista de processos), e o rc fica só em 127.0.0.1.
func TestMountLigaORcSemSegredoNosArgumentos(t *testing.T) {
	g, f, _ := montarParaEnvio(t)
	g.Montagens.mu.Lock()
	info := g.Montagens.montagens[normalizarPonto("V")]
	g.Montagens.mu.Unlock()

	args := strings.Join(f.chamadas()[0].Args, " ")
	if !strings.Contains(args, "--rc --rc-addr 127.0.0.1:") {
		t.Errorf("args = %s", args)
	}
	if strings.Contains(args, info.rc.usuario) || strings.Contains(args, info.rc.senha) {
		t.Error("credenciais do rc nos argumentos")
	}
	// Sem a senha, o rc recusa.
	outro := *info.rc
	outro.senha = "errada"
	var e envioVfs
	if err := outro.chamar("vfs/stats", &e); err == nil {
		t.Error("o rc deveria recusar credencial errada")
	}
}

// Processo que já morreu: não há o que esperar (o cache fica para a próxima
// montagem); a desmontagem segue.
func TestDesmontarComProcessoMortoNaoEsperaEnvio(t *testing.T) {
	g, f, pid := montarParaEnvio(t)
	filaFalsa(f, 0, 3, 0)
	matarProcesso(t, pid)
	if !esperarAte(5*time.Second, func() bool { return !processoVivoNoSO(pid) }) {
		t.Fatal("processo não morreu")
	}

	inicio := time.Now()
	g.Montagens.DesmontarUnidade("V")
	if d := time.Since(inicio); d > 5*time.Second {
		t.Errorf("desmontar esperou %v um envio de processo morto", d)
	}
}

func matarProcesso(t *testing.T, pid int) {
	t.Helper()
	p, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Kill(); err != nil {
		t.Fatal(err)
	}
}

package tray

import "testing"

type itemFalso struct{ marcado, chamado bool }

func (i *itemFalso) Check()   { i.marcado, i.chamado = true, true }
func (i *itemFalso) Uncheck() { i.marcado, i.chamado = false, true }

// A marca de "Abrir ao ligar o computador" segue o estado real, lido de novo
// a cada atualização, nos dois sentidos.
func TestMarcaDoAutoIniciarSegueOEstadoReal(t *testing.T) {
	ligado := false
	item := &itemFalso{marcado: true}
	g := &GerenciadorTray{autoIniciarLigado: func() bool { return ligado }, itemAutoIniciar: item}

	if g.AtualizarAutoIniciar() || item.marcado {
		t.Errorf("desligado: marcado = %v", item.marcado)
	}
	ligado = true
	if !g.AtualizarAutoIniciar() || !item.marcado {
		t.Errorf("ligado: marcado = %v", item.marcado)
	}
	ligado = false
	g.AtualizarAutoIniciar()
	if item.marcado {
		t.Error("desligou de novo, mas a marca ficou")
	}
}

// Antes de o menu existir, atualizar só lê o estado.
func TestAtualizarAutoIniciarSemMenu(t *testing.T) {
	g := &GerenciadorTray{autoIniciarLigado: func() bool { return true }}
	if !g.AtualizarAutoIniciar() {
		t.Error("deveria devolver o estado lido")
	}
}

// O estado vem do sistema por padrão.
func TestEstadoDoAutoIniciarVemDoSistema(t *testing.T) {
	g := NovoGerenciadorTray(nil, nil, nil)
	if g.autoIniciarLigado == nil {
		t.Fatal("sem leitura do estado real")
	}
}

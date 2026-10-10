package gui

import "testing"

// Demanda 031 (cópia aprovada): o campo de senha mostra só "Senha".
func TestCampoSenhaPlaceholder(t *testing.T) {
	d, _, _, _ := novoDialogoSenhaTeste(t, func(string) error { return nil })
	if d.entrySenha.PlaceHolder != "Senha" || TextoCampoSenha != "Senha" {
		t.Errorf("placeholder = %q", d.entrySenha.PlaceHolder)
	}
}

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypto/internal/core"
)

// ResultadoNovoCofre contém o resultado do wizard de criação de cofre.
type ResultadoNovoCofre struct {
	Sucesso  bool
	Provedor *core.Provedor
	Nome     string
	Senha    string
}

// DialogoNovoCofre exibe o wizard de criação de cofre.
func DialogoNovoCofre(janelaPai fyne.Window) ResultadoNovoCofre {
	resultado := make(chan ResultadoNovoCofre, 1)

	// Filtrar provedores (sem local_only para criação)
	var provedores []core.Provedor
	for _, p := range core.Provedores {
		if !p.LocalOnly {
			provedores = append(provedores, p)
		}
	}

	var provedorSelecionado *core.Provedor

	// Campos da aba de senha
	entrySenha := widget.NewPasswordEntry()
	entrySenha.SetPlaceHolder("Mínimo 8 caracteres")

	entrySenha2 := widget.NewPasswordEntry()
	entrySenha2.SetPlaceHolder("Repita a senha")

	entryNome := widget.NewEntry()
	entryNome.SetPlaceHolder("Ex: MeusDocumentos")

	// Container das abas
	containerProvedor := container.NewVBox()
	containerSenha := container.NewVBox()

	var dialogo *widget.PopUp
	tabs := container.NewAppTabs(
		container.NewTabItem("1. Provedor", containerProvedor),
		container.NewTabItem("2. Senha", containerSenha),
	)

	// === Aba Provedores ===
	lblEscolha := canvas.NewText("Escolha o provedor de nuvem:", CorTexto)
	lblEscolha.TextSize = 12
	containerProvedor.Add(lblEscolha)

	for _, prov := range provedores {
		p := prov // captura
		btn := widget.NewButton(p.Nome, func() {
			provedorSelecionado = &p
			tabs.SelectIndex(1)
		})
		btn.Importance = widget.MediumImportance
		containerProvedor.Add(btn)
	}

	// === Aba Senha ===
	lblSenha := canvas.NewText("Senha:", CorTextoSec)
	lblSenha.TextSize = 11

	lblSenha2 := canvas.NewText("Confirmar senha:", CorTextoSec)
	lblSenha2.TextSize = 11

	lblNome := canvas.NewText("Nome do cofre:", CorTextoSec)
	lblNome.TextSize = 11

	btnCancelar := widget.NewButton("Cancelar", func() {
		resultado <- ResultadoNovoCofre{Sucesso: false}
		if dialogo != nil {
			dialogo.Hide()
		}
	})

	btnCriar := widget.NewButton("Criar Cofre", func() {
		if provedorSelecionado == nil {
			DialogoMensagem(janelaPai, "Erro", "Selecione um provedor primeiro.", MsgErro)
			return
		}
		nome := entryNome.Text
		if nome == "" {
			DialogoMensagem(janelaPai, "Erro", "Informe um nome para o cofre.", MsgErro)
			return
		}
		senha := entrySenha.Text
		if len(senha) < 8 {
			DialogoMensagem(janelaPai, "Erro", "A senha deve ter pelo menos 8 caracteres.", MsgErro)
			return
		}
		if senha != entrySenha2.Text {
			DialogoMensagem(janelaPai, "Erro", "As senhas não coincidem.", MsgErro)
			return
		}
		resultado <- ResultadoNovoCofre{
			Sucesso:  true,
			Provedor: provedorSelecionado,
			Nome:     nome,
			Senha:    senha,
		}
		if dialogo != nil {
			dialogo.Hide()
		}
	})
	btnCriar.Importance = widget.HighImportance

	containerSenha.Add(lblSenha)
	containerSenha.Add(entrySenha)
	containerSenha.Add(lblSenha2)
	containerSenha.Add(entrySenha2)
	containerSenha.Add(lblNome)
	containerSenha.Add(entryNome)
	containerSenha.Add(layout.NewSpacer())
	containerSenha.Add(container.NewHBox(btnCancelar, layout.NewSpacer(), btnCriar))

	// === Header ===
	lblIcone := canvas.NewText("🔒", CorVerde)
	lblIcone.TextSize = 32
	lblIcone.Alignment = fyne.TextAlignCenter

	lblTitulo := canvas.NewText("Criar Novo Cofre", CorVerde)
	lblTitulo.TextSize = 18
	lblTitulo.TextStyle = fyne.TextStyle{Bold: true}
	lblTitulo.Alignment = fyne.TextAlignCenter

	conteudo := container.NewVBox(
		lblIcone,
		lblTitulo,
		widget.NewSeparator(),
		tabs,
	)

	padded := container.NewPadded(conteudo)

	dialogo = widget.NewModalPopUp(padded, janelaPai.Canvas())
	dialogo.Resize(fyne.NewSize(500, 520))
	dialogo.Show()

	return <-resultado
}

// ResultadoImportarCofre contém o resultado do wizard de importação.
type ResultadoImportarCofre struct {
	Sucesso  bool
	Provedor *core.Provedor
	Nome     string
	Senha    string
	Senha2   string
}

// DialogoImportarCofre exibe o wizard de importação de cofre existente.
func DialogoImportarCofre(janelaPai fyne.Window) ResultadoImportarCofre {
	resultado := make(chan ResultadoImportarCofre, 1)

	var provedorSelecionado *core.Provedor

	entrySenha := widget.NewPasswordEntry()
	entrySenha.SetPlaceHolder("Senha usada na criação do cofre")

	entrySenha2 := widget.NewPasswordEntry()
	entrySenha2.SetPlaceHolder("Deixe vazio se igual à senha")

	entryNome := widget.NewEntry()
	entryNome.SetPlaceHolder("Ex: MeusDocumentos")

	containerProvedor := container.NewVBox()
	containerDados := container.NewVBox()

	var dialogo *widget.PopUp
	tabs := container.NewAppTabs(
		container.NewTabItem("1. Provedor", containerProvedor),
		container.NewTabItem("2. Senhas", containerDados),
	)

	// === Aba Provedores ===
	lblEscolha := canvas.NewText("Onde está o cofre existente?", CorTexto)
	lblEscolha.TextSize = 12
	containerProvedor.Add(lblEscolha)

	for _, prov := range core.Provedores {
		p := prov
		btn := widget.NewButton(p.Nome, func() {
			provedorSelecionado = &p
			tabs.SelectIndex(1)
		})
		btn.Importance = widget.MediumImportance
		containerProvedor.Add(btn)
	}

	// === Aba Dados ===
	lblSenha := canvas.NewText("Senha do cofre (password):", CorTextoSec)
	lblSenha.TextSize = 11
	lblSenha2 := canvas.NewText("Senha 2 / salt (password2):", CorTextoSec)
	lblSenha2.TextSize = 11
	lblNome := canvas.NewText("Nome para o cofre:", CorTextoSec)
	lblNome.TextSize = 11

	lblDica := canvas.NewText("💡 Após confirmar, você selecionará a pasta\nonde o cofre está na nuvem.", CorTextoSec)
	lblDica.TextSize = 11

	btnCancelar := widget.NewButton("Cancelar", func() {
		resultado <- ResultadoImportarCofre{Sucesso: false}
		if dialogo != nil {
			dialogo.Hide()
		}
	})

	btnAvancar := widget.NewButton("Avançar  →", func() {
		if provedorSelecionado == nil {
			DialogoMensagem(janelaPai, "Erro", "Selecione um provedor primeiro.", MsgErro)
			return
		}
		nome := entryNome.Text
		if nome == "" {
			DialogoMensagem(janelaPai, "Erro", "Informe um nome para o cofre.", MsgErro)
			return
		}
		senha := entrySenha.Text
		if senha == "" {
			DialogoMensagem(janelaPai, "Erro", "Informe a senha do cofre.", MsgErro)
			return
		}
		senha2 := entrySenha2.Text
		if senha2 == "" {
			senha2 = senha
		}
		resultado <- ResultadoImportarCofre{
			Sucesso:  true,
			Provedor: provedorSelecionado,
			Nome:     nome,
			Senha:    senha,
			Senha2:   senha2,
		}
		if dialogo != nil {
			dialogo.Hide()
		}
	})
	btnAvancar.Importance = widget.HighImportance

	containerDados.Add(lblSenha)
	containerDados.Add(entrySenha)
	containerDados.Add(lblSenha2)
	containerDados.Add(entrySenha2)
	containerDados.Add(lblNome)
	containerDados.Add(entryNome)
	containerDados.Add(lblDica)
	containerDados.Add(layout.NewSpacer())
	containerDados.Add(container.NewHBox(btnCancelar, layout.NewSpacer(), btnAvancar))

	// === Header ===
	lblIconeH := canvas.NewText("📥", CorVerde)
	lblIconeH.TextSize = 32
	lblIconeH.Alignment = fyne.TextAlignCenter

	lblTitulo := canvas.NewText("Importar Cofre Existente", CorVerde)
	lblTitulo.TextSize = 18
	lblTitulo.TextStyle = fyne.TextStyle{Bold: true}
	lblTitulo.Alignment = fyne.TextAlignCenter

	conteudo := container.NewVBox(
		lblIconeH,
		lblTitulo,
		widget.NewSeparator(),
		tabs,
	)

	padded := container.NewPadded(conteudo)

	dialogo = widget.NewModalPopUp(padded, janelaPai.Canvas())
	dialogo.Resize(fyne.NewSize(520, 560))
	dialogo.Show()

	return <-resultado
}

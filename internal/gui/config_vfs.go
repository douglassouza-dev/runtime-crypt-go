package gui

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// descricoesVfs contém as descrições em português de cada parâmetro VFS.
var descricoesVfs = map[string]string{
	"vfs_cache_mode":            "Modo de cache (off, minimal, writes, full)",
	"vfs_cache_max_size":        "Tamanho máximo do cache em disco",
	"vfs_cache_max_age":         "Tempo máximo de vida do cache",
	"vfs_read_chunk_size":       "Tamanho do chunk de leitura",
	"vfs_read_chunk_size_limit": "Limite de chunk (0 = ilimitado)",
	"vfs_read_ahead":            "Tamanho de read-ahead",
	"buffer_size":               "Tamanho do buffer de memória",
	"dir_cache_time":            "Tempo de cache de diretórios",
	"poll_interval":             "Intervalo de polling para mudanças",
	"attr_timeout":              "Cache de atributos (evita consultas repetidas)",
	"vfs_write_back":            "Atraso antes de enviar arquivo à nuvem",
	"vfs_disk_space_total_size": "Tamanho total virtual da unidade",
	"cache_dir":                 "Pasta local para cache VFS (SSD recomendado)",
}

// ordemChaves define a ordem de exibição dos parâmetros.
var ordemChaves = []string{
	"vfs_cache_mode", "vfs_cache_max_size", "vfs_cache_max_age",
	"vfs_read_chunk_size", "vfs_read_chunk_size_limit", "vfs_read_ahead",
	"buffer_size", "dir_cache_time", "poll_interval",
	"attr_timeout", "vfs_write_back", "vfs_disk_space_total_size",
	"cache_dir",
}

// formVfs são os campos do painel VFS. Os valores só chegam ao core em
// salvar (demanda 011): "Restaurar Padrões" só troca o texto dos campos.
type formVfs struct {
	vfs     *core.ConfigVfs
	entries map[string]*widget.Entry
	lblErro *widget.Label
}

func novoFormVfs(vfs *core.ConfigVfs) *formVfs {
	f := &formVfs{vfs: vfs, entries: make(map[string]*widget.Entry)}
	cfgAtual := vfs.Obter()
	for _, chave := range ordemChaves {
		entry := widget.NewEntry()
		entry.SetText(cfgAtual[chave])
		f.entries[chave] = entry
	}
	f.lblErro = widget.NewLabel("")
	f.lblErro.Wrapping = fyne.TextWrapWord
	f.lblErro.Importance = widget.DangerImportance
	f.lblErro.Hide()
	return f
}

// restaurarPadroes põe os valores padrão nos campos, sem aplicar.
func (f *formVfs) restaurarPadroes() {
	for chave, entry := range f.entries {
		entry.SetText(core.ConfiguracoesVfsPadrao[chave])
	}
}

// salvar manda os campos ao core. Se algum valor for recusado, nada muda no
// core, o motivo de cada campo aparece no painel e salvar devolve false.
func (f *formVfs) salvar() bool {
	config := make(map[string]string)
	for chave, entry := range f.entries {
		config[chave] = entry.Text
	}
	_, err := f.vfs.Atualizar(config)
	if err == nil {
		f.lblErro.Hide()
		return true
	}

	var erros core.ErrosVfs
	if !errors.As(err, &erros) {
		// Falha ao gravar vfs.json (demanda 012).
		f.lblErro.SetText("Nada foi salvo: " + err.Error())
		f.lblErro.Show()
		return false
	}
	linhas := []string{"Nada foi salvo. Corrija:"}
	for _, chave := range ordemChaves {
		if e := erros[chave]; e != nil {
			desc := descricoesVfs[chave]
			if desc == "" {
				desc = chave
			}
			linhas = append(linhas, desc+": "+e.Error())
		}
	}
	f.lblErro.SetText(strings.Join(linhas, "\n"))
	f.lblErro.Show()
	return false
}

// DialogoConfigVfs exibe o painel de configurações VFS.
func DialogoConfigVfs(janelaPai fyne.Window, gerenciador *core.GerenciadorRClone) {
	if gerenciador.SomenteLeitura() {
		return // 031: Configurações está desabilitado
	}
	resultado := make(chan struct{}, 1)

	form := novoFormVfs(gerenciador.Vfs)

	// Header
	lblIcone := canvas.NewText("⚙️", CorVerde)
	lblIcone.TextSize = 28
	lblIcone.Alignment = fyne.TextAlignCenter

	lblTitulo := canvas.NewText("Configurações VFS", CorVerde)
	lblTitulo.TextSize = 16
	lblTitulo.TextStyle = fyne.TextStyle{Bold: true}
	lblTitulo.Alignment = fyne.TextAlignCenter

	lblSub := canvas.NewText("Ajustes de cache e streaming do RClone", CorTextoSec)
	lblSub.TextSize = 11
	lblSub.Alignment = fyne.TextAlignCenter

	// Campos
	containerCampos := container.NewVBox()
	for _, chave := range ordemChaves {
		desc := descricoesVfs[chave]
		if desc == "" {
			desc = chave
		}

		lblDesc := canvas.NewText(desc, CorTextoSec)
		lblDesc.TextSize = 11

		linha := container.NewGridWithColumns(2, lblDesc, form.entries[chave])
		containerCampos.Add(linha)
	}

	scrollCampos := container.NewVScroll(containerCampos)
	scrollCampos.SetMinSize(fyne.NewSize(0, 260))

	var dialogo *widget.PopUp

	// Botões
	btnRestaurar := widget.NewButton("Restaurar Padrões", form.restaurarPadroes)

	btnCancelar := widget.NewButton("Cancelar", func() {
		resultado <- struct{}{}
		if dialogo != nil {
			dialogo.Hide()
		}
	})

	btnSalvar := widget.NewButton("Salvar", func() {
		if !form.salvar() {
			return // o painel fica aberto com o motivo
		}
		if dialogo != nil {
			dialogo.Hide()
		}
		DialogoMensagem(janelaPai, "VFS", "Configurações VFS salvas com sucesso.", MsgInfo)
		resultado <- struct{}{}
	})
	btnSalvar.Importance = widget.HighImportance

	conteudo := container.NewVBox(
		lblIcone,
		lblTitulo,
		lblSub,
		widget.NewSeparator(),
		scrollCampos,
		form.lblErro,
		layout.NewSpacer(),
		container.NewHBox(btnRestaurar, btnCancelar, layout.NewSpacer(), btnSalvar),
	)

	padded := container.NewPadded(conteudo)

	dialogo = widget.NewModalPopUp(padded, janelaPai.Canvas())
	dialogo.Resize(fyne.NewSize(520, 500))
	dialogo.Show()

	<-resultado
}

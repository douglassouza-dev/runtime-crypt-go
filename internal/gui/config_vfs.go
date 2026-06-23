package gui

import (
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

// DialogoConfigVfs exibe o painel de configurações VFS.
func DialogoConfigVfs(janelaPai fyne.Window, gerenciador *core.GerenciadorRClone) {
	resultado := make(chan struct{}, 1)

	entries := make(map[string]*widget.Entry)
	cfgAtual := gerenciador.Vfs.Obter()

	aplicarPreset := func(preset core.PresetVfs) {
		for chave, entry := range entries {
			if valor, ok := preset.Valores[chave]; ok {
				entry.SetText(valor)
			}
		}
	}

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

	// Seletor de presets
	lblPresets := canvas.NewText("🎯 Presets por tipo de uso:", CorTextoSec)
	lblPresets.TextSize = 11

	containerPresets := container.NewHBox()
	for _, preset := range core.PresetsVfs {
		p := preset
		btn := widget.NewButton(p.Nome, func() { aplicarPreset(p) })
		btn.Importance = widget.LowImportance
		containerPresets.Add(btn)
	}

	containerPresetsScroll := container.NewHScroll(containerPresets)
	containerPresetsScroll.SetMinSize(fyne.NewSize(0, 40))

	// Campos
	containerCampos := container.NewVBox()
	for _, chave := range ordemChaves {
		valor := cfgAtual[chave]
		desc := descricoesVfs[chave]
		if desc == "" {
			desc = chave
		}

		lblDesc := canvas.NewText(desc, CorTextoSec)
		lblDesc.TextSize = 11

		entry := widget.NewEntry()
		entry.SetText(valor)
		entries[chave] = entry

		linha := container.NewGridWithColumns(2, lblDesc, entry)
		containerCampos.Add(linha)
	}

	scrollCampos := container.NewVScroll(containerCampos)
	scrollCampos.SetMinSize(fyne.NewSize(0, 200))

	var dialogo *widget.PopUp

	// Botões
	btnRestaurar := widget.NewButton("Restaurar Padrões", func() {
		gerenciador.Vfs.Restaurar()
		cfg := gerenciador.Vfs.Obter()
		for chave, entry := range entries {
			entry.SetText(cfg[chave])
		}
	})

	btnCancelar := widget.NewButton("Cancelar", func() {
		resultado <- struct{}{}
		if dialogo != nil {
			dialogo.Hide()
		}
	})

	btnSalvar := widget.NewButton("Salvar", func() {
		config := make(map[string]string)
		for chave, entry := range entries {
			config[chave] = entry.Text
		}
		gerenciador.Vfs.Atualizar(config)
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
		lblPresets,
		containerPresetsScroll,
		widget.NewSeparator(),
		scrollCampos,
		layout.NewSpacer(),
		container.NewHBox(btnRestaurar, btnCancelar, layout.NewSpacer(), btnSalvar),
	)

	padded := container.NewPadded(conteudo)

	dialogo = widget.NewModalPopUp(padded, janelaPai.Canvas())
	dialogo.Resize(fyne.NewSize(520, 580))
	dialogo.Show()

	<-resultado
}

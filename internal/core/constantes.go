package core

// ConfiguracoesVfsPadrao contém os valores padrão para cache e streaming do RClone.
var ConfiguracoesVfsPadrao = map[string]string{
	"vfs_cache_mode":             "full",
	"vfs_cache_max_size":         "10G",
	"vfs_cache_max_age":          "1h",
	"vfs_read_chunk_size":        "8M",
	"vfs_read_chunk_size_limit":  "512M",
	"vfs_read_ahead":             "16M",
	"buffer_size":                "16M",
	"dir_cache_time":             "30m",
	"poll_interval":              "30s",
	"attr_timeout":               "1m",
	"vfs_write_back":             "5s",
	"vfs_disk_space_total_size":  "1T",
	"cache_dir":                  "",
}

// PresetVfs agrupa um nome, descrição e valores de configuração VFS para um caso de uso específico.
type PresetVfs struct {
	Nome     string
	Descricao string
	Valores  map[string]string
}

// PresetsVfs contém configurações VFS pré-otimizadas por tipo de conteúdo.
var PresetsVfs = []PresetVfs{
	{
		Nome: "Streaming de Vídeo", Descricao: "Otimizado para reprodução fluida de vídeos grandes — cache agressivo, chunks grandes, muito read-ahead.",
		Valores: map[string]string{
			"vfs_cache_mode":             "full",
			"vfs_cache_max_size":         "50G",
			"vfs_cache_max_age":          "24h",
			"vfs_read_chunk_size":        "128M",
			"vfs_read_chunk_size_limit":  "0",
			"vfs_read_ahead":             "256M",
			"buffer_size":                "32M",
			"dir_cache_time":             "1h",
			"poll_interval":              "1m",
			"attr_timeout":               "1m",
			"vfs_write_back":             "5s",
			"vfs_disk_space_total_size":  "1T",
			"cache_dir":                  "",
		},
	},
	{
		Nome: "Documentos e Escritório", Descricao: "Para edição colaborativa de documentos — cache compacto, polling rápido para detectar mudanças, write-back ágil.",
		Valores: map[string]string{
			"vfs_cache_mode":             "writes",
			"vfs_cache_max_size":         "2G",
			"vfs_cache_max_age":          "1h",
			"vfs_read_chunk_size":        "4M",
			"vfs_read_chunk_size_limit":  "64M",
			"vfs_read_ahead":             "8M",
			"buffer_size":                "8M",
			"dir_cache_time":             "10m",
			"poll_interval":              "10s",
			"attr_timeout":               "30s",
			"vfs_write_back":             "1s",
			"vfs_disk_space_total_size":  "100G",
			"cache_dir":                  "",
		},
	},
	{
		Nome: "Livros e PDFs", Descricao: "Leitura sequencial de arquivos grandes — cache moderado, read-ahead consistente para virar páginas sem atraso.",
		Valores: map[string]string{
			"vfs_cache_mode":             "full",
			"vfs_cache_max_size":         "5G",
			"vfs_cache_max_age":          "4h",
			"vfs_read_chunk_size":        "32M",
			"vfs_read_chunk_size_limit":  "128M",
			"vfs_read_ahead":             "32M",
			"buffer_size":                "8M",
			"dir_cache_time":             "30m",
			"poll_interval":              "30s",
			"attr_timeout":               "1m",
			"vfs_write_back":             "5s",
			"vfs_disk_space_total_size":  "100G",
			"cache_dir":                  "",
		},
	},
	{
		Nome: "Backup", Descricao: "Foco em escrita — cache writes-only, chunks grandes para upload, sem read-ahead para não desperdiçar cache.",
		Valores: map[string]string{
			"vfs_cache_mode":             "writes",
			"vfs_cache_max_size":         "5G",
			"vfs_cache_max_age":          "30m",
			"vfs_read_chunk_size":        "64M",
			"vfs_read_chunk_size_limit":  "256M",
			"vfs_read_ahead":             "0",
			"buffer_size":                "16M",
			"dir_cache_time":             "5m",
			"poll_interval":              "30s",
			"attr_timeout":               "10s",
			"vfs_write_back":             "5s",
			"vfs_disk_space_total_size":  "1T",
			"cache_dir":                  "",
		},
	},
	{
		Nome: "Fotos", Descricao: "Cache minimal para galerias de imagens — acesso eventual, baixo consumo de disco, atributos com cache longo.",
		Valores: map[string]string{
			"vfs_cache_mode":             "minimal",
			"vfs_cache_max_size":         "2G",
			"vfs_cache_max_age":          "1h",
			"vfs_read_chunk_size":        "16M",
			"vfs_read_chunk_size_limit":  "128M",
			"vfs_read_ahead":             "16M",
			"buffer_size":                "8M",
			"dir_cache_time":             "10m",
			"poll_interval":              "1m",
			"attr_timeout":               "10m",
			"vfs_write_back":             "5s",
			"vfs_disk_space_total_size":  "500G",
			"cache_dir":                  "",
		},
	},
}

// ConfiguracoesCryptPadrao contém os valores padrão para criptografia de nomes e dados.
var ConfiguracoesCryptPadrao = map[string]string{
	"filename_encryption":        "standard",
	"directory_name_encryption":  "true",
	"no_data_encryption":         "false",
}

// LetrasPreferidas define a ordem de preferência para letras de unidade no Windows.
var LetrasPreferidas = []string{
	"V", "W", "X", "Y", "Z", "R", "Q", "P", "O", "N",
	"M", "L", "K", "J", "I", "H", "G", "F", "E", "D",
	"C", "B", "A",
}

// ArquivoCofres é o nome do arquivo JSON onde os metadados dos cofres são persistidos.
const ArquivoCofres = "vaults.json"

// Versao atual do programa.
const Versao = "3.1.0"

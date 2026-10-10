package core

// ConfiguracoesVfsPadrao contém os valores padrão para cache e streaming do RClone.
var ConfiguracoesVfsPadrao = map[string]string{
	"vfs_cache_mode":            "full",
	"vfs_cache_max_size":        "10G",
	"vfs_cache_max_age":         "1h",
	"vfs_read_chunk_size":       "8M",
	"vfs_read_chunk_size_limit": "512M",
	"vfs_read_ahead":            "16M",
	"buffer_size":               "16M",
	"dir_cache_time":            "30m",
	"poll_interval":             "30s",
	"attr_timeout":              "1m",
	"vfs_write_back":            "5s",
	"vfs_disk_space_total_size": "1T",
	"cache_dir":                 "",
}

// ConfiguracoesCryptPadrao contém os valores padrão para criptografia de nomes e dados.
var ConfiguracoesCryptPadrao = map[string]string{
	"filename_encryption":       "standard",
	"directory_name_encryption": "true",
	"no_data_encryption":        "false",
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
const Versao = "3.0"

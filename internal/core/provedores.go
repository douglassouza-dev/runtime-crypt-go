package core

// CampoProvedor define um campo de configuração de um provedor (ex: Access Key, Endpoint).
type CampoProvedor struct {
	Id       string `json:"id"`
	Label    string `json:"label"`
	Tipo     string `json:"tipo"`
	Required bool   `json:"required"`
}

// Provedor define um provedor de nuvem suportado.
type Provedor struct {
	Id        string          `json:"id"`
	Nome      string          `json:"nome"`
	Icone     string          `json:"icone"`
	Cor       string          `json:"cor"`
	OAuth     bool            `json:"oauth"`
	LocalOnly bool            `json:"local_only,omitempty"`
	Campos    []CampoProvedor `json:"campos"`
}

// Provedores é a lista de provedores de nuvem suportados pelo programa.
var Provedores = []Provedor{
	{
		Id: "drive", Nome: "Google Drive", Icone: "G",
		Cor: "#34A853", OAuth: true, Campos: nil,
	},
	{
		Id: "onedrive", Nome: "Microsoft OneDrive", Icone: "O",
		Cor: "#0078D4", OAuth: true, Campos: nil,
	},
	{
		Id: "dropbox", Nome: "Dropbox", Icone: "D",
		Cor: "#0061FF", OAuth: true, Campos: nil,
	},
	{
		Id: "s3", Nome: "Amazon S3 / MinIO", Icone: "S",
		Cor: "#FF9900", OAuth: false,
		Campos: []CampoProvedor{
			{Id: "access_key_id", Label: "Access Key ID", Tipo: "text", Required: true},
			{Id: "secret_access_key", Label: "Secret Access Key", Tipo: "password", Required: true},
			{Id: "region", Label: "Regiao (ex: us-east-1)", Tipo: "text", Required: false},
			{Id: "endpoint", Label: "Endpoint customizado (MinIO, etc)", Tipo: "text", Required: false},
		},
	},
	{
		Id: "local_path", Nome: "Pasta Local", Icone: "L",
		Cor: "#10b981", OAuth: false, LocalOnly: true,
		Campos: []CampoProvedor{
			{Id: "_local_path", Label: "Caminho da pasta base criptografada", Tipo: "text", Required: true},
		},
	},
}

// ObterProvedor retorna um provedor pelo ID, ou nil se não encontrado.
func ObterProvedor(id string) *Provedor {
	for i := range Provedores {
		if Provedores[i].Id == id {
			return &Provedores[i]
		}
	}
	return nil
}

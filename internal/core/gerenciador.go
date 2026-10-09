package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// GerenciadorRClone é a struct principal que coordena todas as operações.
type GerenciadorRClone struct {
	Executavel  string
	DiretorioApp string
	Cofres      *GerenciadorCofres
	Montagens   *GerenciadorMontagem
	Senhas      *CacheSenhas
	OAuth       *GerenciadorOAuth
	Vfs         *ConfigVfs
}

// NovoGerenciador cria e inicializa o gerenciador principal.
// Usa o diretório do executável do programa e procura o rclone nele e no PATH.
func NovoGerenciador() *GerenciadorRClone {
	return NovoGerenciadorEm(obterDiretorioApp(), "")
}

// NovoGerenciadorEm cria o gerenciador com o diretório do app e o executável
// do rclone escolhidos por quem chama. Com executavel vazio, o rclone é
// procurado como em NovoGerenciador. É o ponto de troca usado pelos testes
// para apontar para um rclone falso.
func NovoGerenciadorEm(diretorioApp string, executavel string) *GerenciadorRClone {
	g := &GerenciadorRClone{
		DiretorioApp: diretorioApp,
		Senhas:       NovoCacheSenhas(),
		OAuth:        NovoGerenciadorOAuth(),
		Vfs:          NovoConfigVfs(),
	}

	g.Executavel = executavel
	if g.Executavel == "" {
		g.Executavel = g.localizarRclone()
	}
	g.Cofres = NovoGerenciadorCofres(diretorioApp)
	g.Montagens = NovoGerenciadorMontagem(g.Executavel, g.Vfs)

	return g
}

// obterDiretorioApp retorna o diretório onde o executável está.
func obterDiretorioApp() string {
	exe, err := os.Executable()
	if err != nil {
		dir, _ := os.Getwd()
		return dir
	}
	return filepath.Dir(exe)
}

// localizarRclone busca o binário do rclone em caminhos comuns.
func (g *GerenciadorRClone) localizarRclone() string {
	nomeExe := "rclone"
	if runtime.GOOS == "windows" {
		nomeExe = "rclone.exe"
	}

	// Tentar no diretório do app
	caminhoLocal := filepath.Join(g.DiretorioApp, nomeExe)
	if testarRclone(caminhoLocal) {
		return caminhoLocal
	}

	// Tentar no PATH do sistema
	if testarRclone(nomeExe) {
		return nomeExe
	}

	// Tentar caminho alternativo
	if testarRclone("rclone") {
		return "rclone"
	}

	return ""
}

// testarRclone verifica se o binário do rclone funciona.
func testarRclone(caminho string) bool {
	cmd := exec.Command(caminho, "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	return err == nil
}

// EstaDisponivel verifica se o rclone foi encontrado.
func (g *GerenciadorRClone) EstaDisponivel() bool {
	return g.Executavel != ""
}

// ObscurecerSenha ofusca uma senha usando rclone obscure.
func (g *GerenciadorRClone) ObscurecerSenha(senha string) (string, error) {
	if !g.EstaDisponivel() {
		return "", fmt.Errorf("RClone nao disponivel")
	}

	cmd := exec.Command(g.Executavel, "obscure", "-")
	cmd.Stdin = strings.NewReader(senha)
	if runtime.GOOS == "windows" {
		configurarOcultarJanela(cmd)
	}

	saida, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(saida)), nil
}

// CriarRemoto cria um remoto na configuração do rclone.
func (g *GerenciadorRClone) CriarRemoto(nome string, tipo string, params map[string]string) (bool, string) {
	if !g.EstaDisponivel() {
		return false, "RClone nao disponivel."
	}

	args := []string{"config", "create", nome, tipo}
	for chave, valor := range params {
		if valor != "" {
			args = append(args, chave, valor)
		}
	}

	cmd := exec.Command(g.Executavel, args...)
	saida, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(saida))
		if msg == "" {
			msg = "Erro desconhecido."
		}
		return false, msg
	}
	return true, fmt.Sprintf("Remoto '%s' criado com sucesso.", nome)
}

// CriarCrypt cria um remoto crypt com as senhas ofuscadas.
func (g *GerenciadorRClone) CriarCrypt(nome string, remotoBase string, senha string, senha2 string, configCrypt map[string]string) (bool, string) {
	if !g.EstaDisponivel() {
		return false, "RClone nao disponivel."
	}

	cfg := make(map[string]string)
	for k, v := range ConfiguracoesCryptPadrao {
		cfg[k] = v
	}
	if configCrypt != nil {
		for k, v := range configCrypt {
			if v != "" {
				cfg[k] = v
			}
		}
	}

	senhaObs, err := g.ObscurecerSenha(senha)
	if err != nil {
		return false, err.Error()
	}

	var senha2Obs string
	if senha2 != "" {
		senha2Obs, err = g.ObscurecerSenha(senha2)
		if err != nil {
			return false, err.Error()
		}
	} else {
		senha2Obs = senhaObs
	}

	params := map[string]string{
		"remote":                     remotoBase,
		"password":                   senhaObs,
		"password2":                  senha2Obs,
		"filename_encryption":        cfg["filename_encryption"],
		"directory_name_encryption":  cfg["directory_name_encryption"],
	}
	if cfg["no_data_encryption"] == "true" {
		params["no_data_encryption"] = "true"
	}

	return g.CriarRemoto(nome, "crypt", params)
}

// ImportarCrypt importa um cofre crypt existente (mesma lógica de CriarCrypt).
func (g *GerenciadorRClone) ImportarCrypt(nome string, remotoBase string, senha string, senha2 string, configCrypt map[string]string) (bool, string) {
	return g.CriarCrypt(nome, remotoBase, senha, senha2, configCrypt)
}

// RemoverRemoto remove um remoto da configuração do rclone.
func (g *GerenciadorRClone) RemoverRemoto(nome string) (bool, string) {
	if !g.EstaDisponivel() {
		return false, "RClone nao disponivel."
	}

	nomeLimpo := strings.TrimSuffix(nome, ":")
	cmd := exec.Command(g.Executavel, "config", "delete", nomeLimpo)
	if runtime.GOOS == "windows" {
		configurarOcultarJanela(cmd)
	}

	saida, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(saida))
		if msg == "" {
			msg = "Erro ao remover."
		}
		return false, msg
	}
	return true, fmt.Sprintf("Remoto '%s' removido.", nomeLimpo)
}

// ListarRemotos retorna apenas os remotos do tipo crypt.
func (g *GerenciadorRClone) ListarRemotos() []string {
	if !g.EstaDisponivel() {
		return nil
	}

	cmd := exec.Command(g.Executavel, "config", "dump")
	saida, err := cmd.Output()
	if err != nil {
		return nil
	}

	var config map[string]map[string]interface{}
	if err := json.Unmarshal(saida, &config); err != nil {
		return nil
	}

	var remotos []string
	for nome, cfg := range config {
		if tipo, ok := cfg["type"].(string); ok && tipo == "crypt" {
			remotos = append(remotos, nome+":")
		}
	}
	return remotos
}

// ListarTodosRemotos retorna todos os remotos configurados.
func (g *GerenciadorRClone) ListarTodosRemotos() []string {
	if !g.EstaDisponivel() {
		return nil
	}

	cmd := exec.Command(g.Executavel, "listremotes")
	saida, err := cmd.Output()
	if err != nil {
		return nil
	}

	var remotos []string
	for _, linha := range strings.Split(string(saida), "\n") {
		linha = strings.TrimSpace(linha)
		if linha != "" {
			remotos = append(remotos, linha)
		}
	}
	return remotos
}

// RemotoDetalhado contém informações detalhadas de um remoto.
type RemotoDetalhado struct {
	Nome         string `json:"nome"`
	Tipo         string `json:"tipo"`
	IsCrypt      bool   `json:"is_crypt"`
	RemotoBase   string `json:"remoto_base"`
	Montado      bool   `json:"montado"`
	LetraMontada string `json:"letra_montada,omitempty"`
}

// ListarRemotosDetalhado retorna todos os remotos com tipo, config e status de montagem.
func (g *GerenciadorRClone) ListarRemotosDetalhado() []RemotoDetalhado {
	if !g.EstaDisponivel() {
		return nil
	}

	cmd := exec.Command(g.Executavel, "config", "dump")
	saida, err := cmd.Output()
	if err != nil {
		return nil
	}

	var config map[string]map[string]interface{}
	if err := json.Unmarshal(saida, &config); err != nil {
		return nil
	}

	montagensAtivas := g.Montagens.ObterMontagens()
	var lista []RemotoDetalhado

	for nome, cfg := range config {
		tipo := ""
		if t, ok := cfg["type"].(string); ok {
			tipo = t
		}
		remotoBase := ""
		if r, ok := cfg["remote"].(string); ok {
			remotoBase = r
		}

		m, montado := montagensAtivas[nome]
		item := RemotoDetalhado{
			Nome:       nome,
			Tipo:       tipo,
			IsCrypt:    tipo == "crypt",
			RemotoBase: remotoBase,
			Montado:    montado,
		}
		if montado {
			item.LetraMontada = m.Letra
		}
		lista = append(lista, item)
	}
	return lista
}

// ObterConfigRemoto retorna a configuração de um remoto específico.
func (g *GerenciadorRClone) ObterConfigRemoto(nome string) map[string]interface{} {
	if !g.EstaDisponivel() {
		return nil
	}

	nome = strings.TrimSuffix(nome, ":")
	cmd := exec.Command(g.Executavel, "config", "dump")
	saida, err := cmd.Output()
	if err != nil {
		return nil
	}

	var config map[string]map[string]interface{}
	if err := json.Unmarshal(saida, &config); err != nil {
		return nil
	}

	if cfg, ok := config[nome]; ok {
		return cfg
	}
	return nil
}

// ListarDiretoriosRemoto lista subdiretórios de um remoto via rclone lsd.
func (g *GerenciadorRClone) ListarDiretoriosRemoto(nomeRemoto string, caminho string) []string {
	if !g.EstaDisponivel() {
		return nil
	}

	if !strings.HasSuffix(nomeRemoto, ":") {
		nomeRemoto = nomeRemoto + ":"
	}

	alvo := nomeRemoto
	if caminho != "" {
		alvo = nomeRemoto + strings.TrimLeft(caminho, "/")
	}

	cmd := exec.Command(g.Executavel, "lsd", alvo)
	if runtime.GOOS == "windows" {
		configurarOcultarJanela(cmd)
	}

	tipo := "timeout"
	var saida []byte
	done := make(chan struct{})
	go func() {
		var err error
		saida, err = cmd.Output()
		if err != nil {
			tipo = "erro"
		} else {
			tipo = "ok"
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return nil
	}

	if tipo != "ok" {
		return nil
	}

	var dirs []string
	for _, linha := range strings.Split(string(saida), "\n") {
		linha = strings.TrimSpace(linha)
		if linha == "" {
			continue
		}
		// Formato: -1 2024-01-01 00:00:00 -1 nome_pasta
		partes := strings.SplitN(linha, " ", 5)
		if len(partes) >= 5 {
			dirs = append(dirs, strings.TrimSpace(partes[4]))
		} else if len(partes) >= 1 {
			dirs = append(dirs, partes[len(partes)-1])
		}
	}

	// Ordenar
	sort := func(a []string) {
		for i := 0; i < len(a); i++ {
			for j := i + 1; j < len(a); j++ {
				if a[i] > a[j] {
					a[i], a[j] = a[j], a[i]
				}
			}
		}
	}
	sort(dirs)

	return dirs
}

// ListarCofres retorna todos os cofres com status enriquecido.
func (g *GerenciadorRClone) ListarCofres() []CofreStatus {
	montagens := g.Montagens.ObterMontagens()
	return g.Cofres.Listar(montagens, g.Senhas)
}

// Encerrar limpa recursos ao sair do programa.
func (g *GerenciadorRClone) Encerrar() {
	g.Montagens.DesmontarTodas()
	g.Senhas.LimparTodas()
	g.OAuth.Abortar()
}

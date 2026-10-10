package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// GerenciadorRClone é a struct principal que coordena todas as operações.
type GerenciadorRClone struct {
	Executavel string
	// DiretorioApp é a pasta do executável: o rclone é procurado nela.
	DiretorioApp string
	// DiretorioConfig é onde ficam vaults.json, vfs.json e o log (031).
	DiretorioConfig string
	// ErroPastaConfig (*ErroPastaConfig) vem preenchido quando a pasta de
	// configuração não pode ser usada: nada é gravado e a tela avisa (031).
	ErroPastaConfig error
	// RegistroPastaConfig são as linhas do log sobre a pasta e a cópia dos
	// arquivos antigos; main grava depois de abrir o log.
	RegistroPastaConfig []string
	Cofres              *GerenciadorCofres
	Montagens           *GerenciadorMontagem
	Senhas              *CacheSenhas
	OAuth               *GerenciadorOAuth
	Vfs                 *ConfigVfs

	// ErroCofres vem preenchido quando vaults.json existe mas não pôde ser
	// lido. A tela mostra este erro na abertura (demanda 007).
	ErroCofres error
}

// NovoGerenciador cria e inicializa o gerenciador principal. Procura o rclone
// na pasta do executável e no PATH; vaults.json e vfs.json ficam na pasta de
// configuração do usuário, copiados uma vez da pasta do executável (031).
func NovoGerenciador() *GerenciadorRClone {
	return novoGerenciadorPadrao(obterDiretorioApp(), "")
}

// novoGerenciadorPadrao é NovoGerenciador com a pasta do executável e o rclone
// escolhidos (testes).
func novoGerenciadorPadrao(dirApp, executavel string) *GerenciadorRClone {
	p := PrepararPastaConfig(dirApp)
	g := novoGerenciador(dirApp, p.Leitura[ArquivoCofres], p.Leitura[ArquivoVfs], executavel, p.Err)
	g.DiretorioConfig = p.Dir
	g.RegistroPastaConfig = p.Registro
	g.ErroPastaConfig = p.Err
	return g
}

// NovoGerenciadorEm cria o gerenciador com o diretório do app e o executável
// do rclone escolhidos por quem chama. Com executavel vazio, o rclone é
// procurado como em NovoGerenciador. É o ponto de troca usado pelos testes
// para apontar para um rclone falso.
//
// Aqui a mesma pasta serve para o rclone, vaults.json, vfs.json e o log, sem
// cópia de arquivos antigos (031 fica em NovoGerenciador).
func NovoGerenciadorEm(diretorioApp string, executavel string) *GerenciadorRClone {
	g := novoGerenciador(diretorioApp, diretorioApp, diretorioApp, executavel, nil)
	g.DiretorioConfig = diretorioApp
	return g
}

// novoGerenciador lê vaults.json de dirCofres e vfs.json de dirVfs. Com
// bloqueio, nada é gravado em lugar nenhum (nem a cópia .corrompido).
func novoGerenciador(dirApp, dirCofres, dirVfs, executavel string, bloqueio error) *GerenciadorRClone {
	g := &GerenciadorRClone{
		DiretorioApp: dirApp,
		Senhas:       NovoCacheSenhas(),
		OAuth:        NovoGerenciadorOAuth(),
		Vfs:          NovoConfigVfsEm(dirVfs),
	}

	g.Executavel = executavel
	if g.Executavel == "" {
		g.Executavel = g.localizarRclone()
	}
	g.Cofres, g.ErroCofres = novoGerenciadorCofres(dirCofres, bloqueio)
	if bloqueio != nil {
		g.Vfs.BloquearGravacao(bloqueio)
	}
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
	return verificarRclone(caminho) == nil
}

// verificarRclone roda `rclone --version` com tempo limite.
func verificarRclone(caminho string) error {
	_, err := chamadaRclone{executavel: caminho, args: []string{"--version"}, limite: limiteVersao}.rodar()
	return err
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

	saida, err := chamadaRclone{
		executavel: g.Executavel,
		args:       []string{"obscure", "-"},
		limite:     limiteObscure,
		stdin:      strings.NewReader(senha),
		ocultar:    true,
	}.rodar()
	if err != nil {
		return "", erroComMotivoDoRclone(err)
	}
	return strings.TrimSpace(string(saida)), nil
}

// CriarRemoto cria um remoto na configuração do rclone. A mensagem de falha
// é a frase da tela (demanda 027); o texto do rclone vai para o log.
func (g *GerenciadorRClone) CriarRemoto(nome string, tipo string, params map[string]string) (bool, string) {
	if err := g.criarRemoto(nome, tipo, params); err != nil {
		return false, err.Error()
	}
	return true, fmt.Sprintf("Remoto '%s' criado com sucesso.", nome)
}

// CriarCrypt cria um remoto crypt com as senhas ofuscadas.
func (g *GerenciadorRClone) CriarCrypt(nome string, remotoBase string, senha string, senha2 string, configCrypt map[string]string) (bool, string) {
	if err := g.criarCrypt(nome, remotoBase, senha, senha2, configCrypt); err != nil {
		return false, err.Error()
	}
	return true, fmt.Sprintf("Remoto '%s' criado com sucesso.", nome)
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
	saida, err := chamadaRclone{
		executavel: g.Executavel,
		args:       []string{"config", "delete", nomeLimpo},
		limite:     limiteConfigLocal,
		combinada:  true,
		ocultar:    true,
	}.rodar()
	if err != nil {
		if errors.Is(err, ErrTempoEsgotado) {
			return false, err.Error()
		}
		if strings.TrimSpace(string(saida)) == "" {
			return false, "Erro ao remover."
		}
		return false, novoErroRclone("config delete", string(saida), err).Error()
	}
	return true, fmt.Sprintf("Remoto '%s' removido.", nomeLimpo)
}

// ErrRcloneIndisponivel é devolvido pelas listagens quando o rclone não foi
// encontrado.
var ErrRcloneIndisponivel = errors.New("rclone nao disponivel")

// ErrRemotoNaoEncontrado é devolvido por ObterConfigRemoto quando o remoto não
// existe no rclone.conf.
var ErrRemotoNaoEncontrado = errors.New("remoto nao encontrado")

// configDump roda `rclone config dump` com tempo limite e devolve o JSON lido.
func (g *GerenciadorRClone) configDump() (map[string]map[string]interface{}, error) {
	saida, err := chamadaRclone{
		executavel: g.Executavel,
		args:       []string{"config", "dump"},
		limite:     limiteConfigLocal,
	}.rodar()
	if err != nil {
		return nil, erroComMotivoDoRclone(err)
	}
	var config map[string]map[string]interface{}
	if err := json.Unmarshal(saida, &config); err != nil {
		return nil, fmt.Errorf("saida de `rclone config dump` ilegivel: %w", err)
	}
	return config, nil
}

// ListarRemotos retorna apenas os remotos do tipo crypt. Erro do rclone volta
// como erro, nunca como lista vazia (demanda 009).
func (g *GerenciadorRClone) ListarRemotos() ([]string, error) {
	if !g.EstaDisponivel() {
		return nil, ErrRcloneIndisponivel
	}

	config, err := g.configDump()
	if err != nil {
		return nil, err
	}

	remotos := []string{}
	for nome, cfg := range config {
		if tipo, ok := cfg["type"].(string); ok && tipo == "crypt" {
			remotos = append(remotos, nome+":")
		}
	}
	return remotos, nil
}

// ListarTodosRemotos retorna todos os remotos configurados.
func (g *GerenciadorRClone) ListarTodosRemotos() ([]string, error) {
	if !g.EstaDisponivel() {
		return nil, ErrRcloneIndisponivel
	}
	return g.listarTodosRemotos()
}

// listarTodosRemotos roda `rclone listremotes` com tempo limite.
func (g *GerenciadorRClone) listarTodosRemotos() ([]string, error) {
	saida, err := chamadaRclone{
		executavel: g.Executavel,
		args:       []string{"listremotes"},
		limite:     limiteConfigLocal,
	}.rodar()
	if err != nil {
		return nil, erroComMotivoDoRclone(err)
	}

	remotos := []string{}
	for _, linha := range strings.Split(string(saida), "\n") {
		linha = strings.TrimSpace(linha)
		if linha != "" {
			remotos = append(remotos, linha)
		}
	}
	return remotos, nil
}

// RemotoDetalhado contém informações detalhadas de um remoto.
type RemotoDetalhado struct {
	Nome         string         `json:"nome"`
	Tipo         string         `json:"tipo"`
	IsCrypt      bool           `json:"is_crypt"`
	RemotoBase   string         `json:"remoto_base"`
	Estado       EstadoMontagem `json:"estado"`
	LetraMontada string         `json:"letra_montada,omitempty"`
}

// ListarRemotosDetalhado retorna todos os remotos com tipo, config e status de montagem.
func (g *GerenciadorRClone) ListarRemotosDetalhado() ([]RemotoDetalhado, error) {
	if !g.EstaDisponivel() {
		return nil, ErrRcloneIndisponivel
	}

	config, err := g.configDump()
	if err != nil {
		return nil, err
	}

	montagensAtivas := g.Montagens.ObterMontagens()
	lista := []RemotoDetalhado{}

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
			Estado:     EstadoDesmontado,
		}
		if montado {
			item.Estado = EstadoMontado
			item.LetraMontada = m.Letra
		}
		lista = append(lista, item)
	}
	return lista, nil
}

// ObterConfigRemoto retorna a configuração de um remoto específico. Remoto
// inexistente devolve ErrRemotoNaoEncontrado.
func (g *GerenciadorRClone) ObterConfigRemoto(nome string) (map[string]interface{}, error) {
	if !g.EstaDisponivel() {
		return nil, ErrRcloneIndisponivel
	}

	nome = strings.TrimSuffix(nome, ":")
	config, err := g.configDump()
	if err != nil {
		return nil, err
	}

	if cfg, ok := config[nome]; ok {
		return cfg, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrRemotoNaoEncontrado, nome)
}

// ListarDiretoriosRemoto lista os subdiretórios de um remoto com o nome exato
// de cada um (demanda 021). Usa `rclone lsjson --dirs-only`, que entrega o
// nome em JSON (campo Name), em vez de recortar as colunas alinhadas do `lsd`.
//
// Pasta vazia devolve lista vazia e nil. Erro do rclone, tempo esgotado ou
// saída ilegível devolvem erro com o motivo (demanda 009).
func (g *GerenciadorRClone) ListarDiretoriosRemoto(nomeRemoto string, caminho string) ([]string, error) {
	if !g.EstaDisponivel() {
		return nil, ErrRcloneIndisponivel
	}

	if !strings.HasSuffix(nomeRemoto, ":") {
		nomeRemoto = nomeRemoto + ":"
	}

	alvo := nomeRemoto
	if caminho != "" {
		alvo = nomeRemoto + strings.TrimLeft(caminho, "/")
	}

	saida, err := chamadaRclone{
		executavel: g.Executavel,
		args:       []string{"lsjson", "--dirs-only", alvo},
		limite:     limiteListagem,
		ocultar:    true,
	}.rodar()
	if err != nil {
		return nil, erroComMotivoDoRclone(err)
	}

	dirs, err := lerNomesLsjson(saida)
	if err != nil {
		return nil, fmt.Errorf("saida de `rclone lsjson` ilegivel: %w", err)
	}

	sort.Strings(dirs)
	return dirs, nil
}

// itemLsjson é o pedaço de `rclone lsjson` que interessa ao seletor.
type itemLsjson struct {
	Name  string `json:"Name"`
	IsDir bool   `json:"IsDir"`
}

// lerNomesLsjson devolve o nome de cada pasta, sem mexer em nenhum caractere.
func lerNomesLsjson(saida []byte) ([]string, error) {
	var itens []itemLsjson
	if err := json.Unmarshal(saida, &itens); err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(itens))
	for _, it := range itens {
		if it.IsDir && it.Name != "" {
			dirs = append(dirs, it.Name)
		}
	}
	return dirs, nil
}

// ListarCofres retorna todos os cofres com status enriquecido.
func (g *GerenciadorRClone) ListarCofres() []CofreStatus {
	return g.Cofres.Listar(g.Montagens.EstadosPorRemoto(), g.Senhas)
}

// Encerrar limpa recursos ao sair do programa.
func (g *GerenciadorRClone) Encerrar() {
	g.Montagens.DesmontarTodas()
	g.Senhas.LimparTodas()
	g.OAuth.Abortar()
}

package core

import (
	"errors"
	"sort"
)

// Fica fora de gerenciador.go pela regra da demanda 009 (lá não há
// "return nil"): aqui nil é o sucesso de funções que só devolvem error.

// criarRemoto é CriarRemoto com o erro tipado: *ErroRclone quando o rclone
// recusou.
func (g *GerenciadorRClone) criarRemoto(nome string, tipo string, params map[string]string) error {
	if !g.EstaDisponivel() {
		return errors.New("RClone nao disponivel.")
	}

	args := argsConfigCreate(nome, tipo, params)

	saida, err := chamadaRclone{
		executavel: g.Executavel,
		args:       args,
		limite:     limiteConfigCreate,
		combinada:  true,
	}.rodar()
	if err != nil {
		if errors.Is(err, ErrTempoEsgotado) {
			return err
		}
		return novoErroRclone("config create "+tipo, string(saida), err)
	}
	return nil
}

func (g *GerenciadorRClone) criarCrypt(nome string, remotoBase string, senha string, senha2 string, configCrypt map[string]string) error {
	if !g.EstaDisponivel() {
		return errors.New("RClone nao disponivel.")
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
		return err
	}

	var senha2Obs string
	if senha2 != "" {
		senha2Obs, err = g.ObscurecerSenha(senha2)
		if err != nil {
			return err
		}
	} else {
		senha2Obs = senhaObs
	}

	params := map[string]string{
		"remote":                    remotoBase,
		"password":                  senhaObs,
		"password2":                 senha2Obs,
		"filename_encryption":       cfg["filename_encryption"],
		"directory_name_encryption": cfg["directory_name_encryption"],
	}
	if cfg["no_data_encryption"] == "true" {
		params["no_data_encryption"] = "true"
	}

	return g.criarRemoto(nome, "crypt", params)
}

// argsConfigCreate monta `config create -- <nome> <tipo> <chave> <valor> …`.
//
// Demanda 029: depois de `--` o rclone não lê mais nada como opção. Sem ele,
// uma senha ofuscada que começa com "-" (cerca de 1 em 64) vira
// "unknown shorthand flag" e a criação do cofre falha. As chaves saem em
// ordem alfabética para a linha de comando ser sempre a mesma.
func argsConfigCreate(nome, tipo string, params map[string]string) []string {
	chaves := make([]string, 0, len(params))
	for chave, valor := range params {
		if valor != "" {
			chaves = append(chaves, chave)
		}
	}
	sort.Strings(chaves)

	args := []string{"config", "create", "--", nome, tipo}
	for _, chave := range chaves {
		args = append(args, chave, params[chave])
	}
	return args
}

package core

import (
	"log"
	"sort"
	"strings"
)

// Demanda 028: ao sair, cofres que caíram com o rclone parado podem ter
// arquivos no cache da VFS que ainda não subiram (contagem da 026).

// PendenciaCofre é um cofre que caiu e ainda tem N arquivos para subir.
type PendenciaCofre struct {
	Nome string
	N    int
}

// PendentesAoSair lista, em ordem de nome, os cofres que caíram com o rclone
// parado e têm arquivo no cache que não subiu. Cofre que caiu com o rclone
// vivo fica de fora: Encerrar já espera o envio dele (025). Se não dá para
// ler a pasta do cache, o cofre fica de fora e o motivo vai para o log (não
// há frase aprovada para "não sei quantos").
func (g *GerenciadorRClone) PendentesAoSair() []PendenciaCofre {
	var lista []PendenciaCofre
	for nome, e := range g.Montagens.EstadosPorRemoto() {
		if e.Estado != EstadoFalhou || !e.Caiu {
			continue
		}
		n, vivo, err := g.Montagens.pendentesDaQueda(e.Letra)
		if err != nil {
			log.Printf("sair: não deu para contar os arquivos de %s que não subiram: %v", nome, err)
			continue
		}
		if vivo || n == 0 {
			continue
		}
		lista = append(lista, PendenciaCofre{Nome: nome, N: n})
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Nome < lista[j].Nome })
	return lista
}

// EtapaEnvio diz em que ponto o "Enviar agora" parou.
type EtapaEnvio string

const (
	EtapaDestrancar EtapaEnvio = "destrancar"
	EtapaTrancar    EtapaEnvio = "trancar"
)

// FalhaEnvio é um cofre que "Enviar agora" não conseguiu trancar.
type FalhaEnvio struct {
	Nome  string
	Etapa EtapaEnvio
	Err   error
}

// EnviarPendentes destranca de novo cada cofre (o rclone retoma os envios
// do cache), espera o envio e tranca (025). Devolve uma falha por cofre que
// não chegou a trancado; cofre que falha fica como está, com a senha.
func (g *GerenciadorRClone) EnviarPendentes(nomes []string, pedirSenha func(nome string) (string, bool)) []FalhaEnvio {
	var falhas []FalhaEnvio
	for _, nome := range nomes {
		if _, err := g.Destrancar(nome, func() (string, bool) { return pedirSenha(nome) }); err != nil {
			falhas = append(falhas, FalhaEnvio{Nome: nome, Etapa: EtapaDestrancar, Err: err})
			continue
		}
		if err := g.Trancar(nome); err != nil {
			falhas = append(falhas, FalhaEnvio{Nome: nome, Etapa: EtapaTrancar, Err: err})
		}
	}
	return falhas
}

// pendentesDaQueda conta o que não subiu da montagem caída na letra. vivo diz
// que o rclone dela ainda roda (aí a contagem do disco não vale).
func (g *GerenciadorMontagem) pendentesDaQueda(letra string) (n int, vivo bool, err error) {
	letra = normalizarPonto(letra)
	g.mu.Lock()
	info, existe := g.montagens[letra]
	g.mu.Unlock()
	if !existe {
		return 0, false, nil
	}
	if info.vivo() {
		return 0, true, nil
	}
	n, err = pendentesNoCacheVfs(info.cacheVfs, strings.TrimSuffix(info.Remoto, ":"))
	return n, false, err
}

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Demanda 026: com o rclone morto (cofre que caiu) não há rc para perguntar o
// que falta enviar. O que falta está no cache da VFS, no disco: cada arquivo
// tem um JSON em {cache}/vfsMeta/{remoto}/{caminho} com "Dirty": true
// enquanto não subiu. É o mesmo registro que o rclone lê ao montar de novo
// para retomar o envio ("vfs cache: queuing for upload").

// pastaCacheRclone é a pasta de cache que o rclone usa com estes argumentos e
// este ambiente: --cache-dir, senão RCLONE_CACHE_DIR, senão a pasta de cache
// do usuário + "rclone" (a mesma regra de makeCacheDir no rclone).
func pastaCacheRclone(args, env []string) string {
	for i, a := range args {
		if a == "--cache-dir" && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(a, "--cache-dir="); ok {
			return v
		}
	}
	for i := len(env) - 1; i >= 0; i-- {
		if v, ok := strings.CutPrefix(env[i], "RCLONE_CACHE_DIR="); ok && v != "" {
			return v
		}
	}
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "rclone")
}

// metaVfs é a parte do JSON de vfsMeta que interessa aqui.
type metaVfs struct {
	Dirty bool `json:"Dirty"`
}

// pendentesNoCacheVfs conta os arquivos do remoto que ainda não subiram. Um
// JSON que não dá para ler conta como pendente: na dúvida, não tranca (o
// rclone, ao montar de novo, resolve o registro).
func pendentesNoCacheVfs(cache, remoto string) (int, error) {
	raiz := filepath.Join(cache, "vfsMeta", strings.TrimSuffix(remoto, ":"))
	n := 0
	err := filepath.WalkDir(raiz, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil {
			if caminho == raiz && errors.Is(err, fs.ErrNotExist) {
				return fs.SkipAll
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		dados, err := os.ReadFile(caminho)
		if err != nil {
			n++
			return nil
		}
		var m metaVfs
		if json.Unmarshal(dados, &m) != nil || m.Dirty {
			n++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ErroNaoSubiram: Trancar recusou porque o cache ainda tem arquivos que não
// chegaram à nuvem (demanda 026). O texto já é o da tela.
type ErroNaoSubiram struct{ N int }

func (e *ErroNaoSubiram) Error() string {
	if e.N == 1 {
		return "1 arquivo ainda não subiu"
	}
	return fmt.Sprintf("%s ainda não subiram", Arquivos(e.N))
}

// DesmontarQueda tranca uma montagem que caiu (demanda 026). Com o rclone
// vivo (só o ponto sumiu ou não responde) vale a 025: DesmontarUnidade
// espera o envio pelo rc. Com o rclone morto, confere o cache no disco: se
// há arquivo que não subiu, não desmonta e devolve *ErroNaoSubiram.
func (g *GerenciadorMontagem) DesmontarQueda(letra string) error {
	letra = normalizarPonto(letra)
	g.mu.Lock()
	info, existe := g.montagens[letra]
	g.mu.Unlock()
	if !existe {
		return fmt.Errorf("Nenhuma montagem ativa em %s", TextoPonto(g.caminhoPonto(letra)))
	}
	if !info.vivo() {
		n, err := pendentesNoCacheVfs(info.cacheVfs, info.Remoto)
		if err != nil {
			return fmt.Errorf("não deu para conferir os arquivos que faltam subir (%v)", err)
		}
		if n > 0 {
			return &ErroNaoSubiram{N: n}
		}
	}
	if ok, msg := g.DesmontarUnidade(letra); !ok {
		return errors.New(msg)
	}
	return nil
}

// temMontagem diz se o remoto ainda tem montagem no mapa (montada ou caída).
func (g *GerenciadorMontagem) temMontagem(remoto string) bool {
	remoto = strings.TrimSuffix(remoto, ":")
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, info := range g.montagens {
		if strings.TrimSuffix(info.Remoto, ":") == remoto {
			return true
		}
	}
	return false
}

//go:build !windows

package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// pontoAtivo diz se a pasta é um ponto de montagem ativo (demanda 013). A
// pasta existe antes de montar; o que muda é o dispositivo: o da pasta passa
// a ser o do FUSE, diferente do da pasta-mãe. Uma montagem presa (rclone
// morto) responde com erro diferente de "não existe" e conta como ativa.
func pontoAtivo(caminho string) bool {
	info, err := os.Lstat(caminho)
	if err != nil {
		return !errors.Is(err, os.ErrNotExist)
	}
	mae, err := os.Lstat(filepath.Dir(filepath.Clean(caminho)))
	if err != nil {
		return false
	}
	a, ok1 := info.Sys().(*syscall.Stat_t)
	b, ok2 := mae.Sys().(*syscall.Stat_t)
	if !ok1 || !ok2 {
		return false
	}
	return a.Dev != b.Dev
}

// limiteDesmontarPonto é o tempo limite de fusermount/umount.
const limiteDesmontarPonto = 10 * time.Second

// desmontarPontoFuse solta uma montagem FUSE presa: fusermount3 -u ou
// fusermount -u no Linux, umount no macOS.
func desmontarPontoFuse(caminho string) error {
	var tentativas [][]string
	if runtime.GOOS == "darwin" {
		tentativas = [][]string{{"umount", caminho}}
	} else {
		tentativas = [][]string{{"fusermount3", "-u", caminho}, {"fusermount", "-u", caminho}}
	}
	var erros []string
	for _, t := range tentativas {
		ctx, cancelar := context.WithTimeout(context.Background(), limiteDesmontarPonto)
		saida, err := exec.CommandContext(ctx, t[0], t[1:]...).CombinedOutput()
		cancelar()
		if err == nil {
			return nil
		}
		erros = append(erros, fmt.Sprintf("%s: %v %s", t[0], err, strings.TrimSpace(string(saida))))
	}
	return errors.New(strings.Join(erros, "; "))
}

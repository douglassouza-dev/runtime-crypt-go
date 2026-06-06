package core

import (
	"testing"
)

func TestCacheSenhas(t *testing.T) {
	cache := NovoCacheSenhas()

	// Testar armazenamento e obtenção
	nomeCofre := "meu_cofre_teste"
	senhaOriginal := "SenhaSuperSegura123!"

	cache.Armazenar(nomeCofre, senhaOriginal)

	if !cache.Existe(nomeCofre) {
		t.Errorf("Esperava que o cofre %s existisse no cache", nomeCofre)
	}

	senhaObtida := cache.Obter(nomeCofre)
	if senhaObtida != senhaOriginal {
		t.Errorf("Esperava obter a senha '%s', mas obteve '%s'", senhaOriginal, senhaObtida)
	}

	// Testar limpeza de um cofre específico
	cache.Limpar(nomeCofre)
	if cache.Existe(nomeCofre) {
		t.Errorf("Esperava que o cofre %s não existisse no cache após limpeza", nomeCofre)
	}

	if cache.Obter(nomeCofre) != "" {
		t.Errorf("Esperava obter string vazia após a limpeza, mas obteve algo")
	}

	// Testar limpeza de tudo
	cache.Armazenar("cofre1", "senha1")
	cache.Armazenar("cofre2", "senha2")

	cache.LimparTodas()

	if cache.Existe("cofre1") || cache.Existe("cofre2") {
		t.Errorf("Esperava que todas as senhas fossem limpas após LimparTodas")
	}
}

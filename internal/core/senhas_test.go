package core

import (
	"testing"
)

func TestCacheSenhas(t *testing.T) {
	cache := NovoCacheSenhas()

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

	cache.Limpar(nomeCofre)
	if cache.Existe(nomeCofre) {
		t.Errorf("Esperava que o cofre %s não existisse no cache após limpeza", nomeCofre)
	}

	if cache.Obter(nomeCofre) != "" {
		t.Errorf("Esperava obter string vazia após a limpeza, mas obteve algo")
	}

	cache.Armazenar("cofre1", "senha1")
	cache.Armazenar("cofre2", "senha2")

	cache.LimparTodas()

	if cache.Existe("cofre1") || cache.Existe("cofre2") {
		t.Errorf("Esperava que todas as senhas fossem limpas após LimparTodas")
	}
}

func TestCacheSenhasZeroizacao(t *testing.T) {
	cache := NovoCacheSenhas()

	senha := "SenhaQueSeraZeroizada!"
	cache.Armazenar("teste", senha)

	if !cache.Existe("teste") {
		t.Fatal("Senha deveria existir antes da zeroização")
	}

	cache.Limpar("teste")

	if cache.Existe("teste") {
		t.Error("Senha não deveria existir após Limpar")
	}

	if obtida := cache.Obter("teste"); obtida != "" {
		t.Errorf("Obter deveria retornar string vazia após Limpar, retornou '%s'", obtida)
	}
}

func TestCacheSenhasZeroizacaoMultipla(t *testing.T) {
	cache := NovoCacheSenhas()

	cache.Armazenar("a", "senhaA")
	cache.Armazenar("b", "senhaB")
	cache.Armazenar("c", "senhaC")

	if !cache.Existe("a") || !cache.Existe("b") || !cache.Existe("c") {
		t.Fatal("Todas as senhas deveriam existir antes da zeroização")
	}

	cache.LimparTodas()

	if cache.Existe("a") || cache.Existe("b") || cache.Existe("c") {
		t.Error("Nenhuma senha deveria existir após LimparTodas")
	}

	if cache.Obter("a") != "" || cache.Obter("b") != "" || cache.Obter("c") != "" {
		t.Error("Obter deveria retornar vazio para todos após LimparTodas")
	}
}

func TestCacheSenhasReutilizacaoAposLimpeza(t *testing.T) {
	cache := NovoCacheSenhas()

	cache.Armazenar("cofre", "primeiraSenha")
	cache.Limpar("cofre")
	cache.Armazenar("cofre", "segundaSenha")

	if obtida := cache.Obter("cofre"); obtida != "segundaSenha" {
		t.Errorf("Esperava 'segundaSenha', obteve '%s'", obtida)
	}
}

func TestCacheSenhasConcorrenciaBasica(t *testing.T) {
	cache := NovoCacheSenhas()
	done := make(chan struct{})

	go func() {
		for i := 0; i < 100; i++ {
			cache.Armazenar("concorrente", "senha")
			cache.Obter("concorrente")
			cache.Existe("concorrente")
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Armazenar("concorrente", "outra")
			cache.Limpar("concorrente")
		}
		done <- struct{}{}
	}()

	<-done
	<-done

	cache.LimparTodas()
}

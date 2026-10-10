# 010 — Ninguém confere a montagem depois que ela sobe

- Estado: Feita (52f93e7)
- Risco: Médio — estabilidade
- Onde: `internal/core/montagem.go:Status`, `MontarUnidade`
- Depende de: 001

## Contexto

O ponto de montagem (`X:\`) só é testado durante a espera de 45 s em `MontarUnidade`. Depois disso, o `core` não confere mais nada. Se o WinFsp derrubar a unidade ou o rclone travar sem sair, o programa continua dizendo que está montado. Isso vale depois que a demanda 001 corrigir o rastreio do processo.

A regra de estado pede: **montado = processo vivo E ponto de montagem existe.**

## O que muda

- `Status` (ou o que a substituir) calcula `montado` com as duas condições.
- Quando uma delas falha, a montagem passa a "falhou" com o motivo ("processo terminou" ou "ponto de montagem sumiu"). Não some do mapa em silêncio.
- A conferência do ponto usa limite de tempo, porque `os.Stat` em unidade travada pode bloquear. O comportamento real do `os.Stat` em WinFsp travado é desconhecido. A implementação mede e registra.

## O que fica de fora

- Remontar sozinho.
- Mostrar o estado na tela (018).

## Pronto quando

- [ ] Teste: processo vivo e ponto inexistente → estado `falhou`, motivo "ponto de montagem sumiu".
- [ ] Teste: processo morto e ponto existente → estado `falhou`, motivo "processo terminou".
- [ ] Teste: os dois válidos → `montado`.
- [ ] O ponto de montagem é injetável no teste (pasta temporária), sem WinFsp.

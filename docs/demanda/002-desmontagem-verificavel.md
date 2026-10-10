# 002 — Trancar informa sucesso sem confirmar a desmontagem

- Estado: Feita (4b3d378)
- Risco: Alto — estabilidade e segredos
- Onde: `internal/core/montagem.go:DesmontarUnidade`; `main.go:travarCofre`
- Depende de: 001

## Contexto

`DesmontarUnidade` tenta três vezes: `Interrupt` e depois `Kill` duas vezes, com 5 s de espera cada. Depois das três, vai para `limpeza` de qualquer jeito, apaga a letra do mapa e devolve "desmontada com sucesso". A cada tentativa, uma nova goroutine chama `Process.Wait()`. Ou seja, `Wait` pode ser chamado mais de uma vez no mesmo processo.

No Windows, `Signal(os.Interrupt)` falha na hora. O primeiro ciclo gasta 5 s à toa antes do `Kill`. Um `Kill` não dá chance ao rclone de enviar o que está no `--vfs-write-back` (padrão 5 s). Se o rclone parar antes de terminar esse envio, os arquivos podem não chegar ao provedor. Quanto se perde em cada caso é desconhecido.

`travarCofre` chama `DesmontarUnidade(nome)` quando não acha a letra. Passa o nome do cofre no lugar da letra.

O usuário vê "Cofre Trancado" e a senha sai do cache, mas a unidade pode continuar aberta.

## O que muda

- A desmontagem pede o encerramento normal do rclone primeiro. No Windows, o caminho exato fica a definir na implementação, com o motivo escrito no PR. Opções conhecidas: `rclone rc` desligado aqui, sinal de console ou `Kill`.
- Espera o fim do processo pela mesma goroutine de `Wait` da demanda 001.
- Só informa sucesso quando o processo terminou **e** o ponto de montagem sumiu. Fora isso, devolve erro com o motivo.
- `travarCofre` não passa mais o nome do cofre como letra.

## O que fica de fora

- Esperar o fim do envio do write-back antes de desmontar. Se for preciso, vira outra demanda.
- Mudanças na tela além da mensagem de erro.

## Pronto quando

- [ ] Teste em `internal/core`: um processo que ignora `Interrupt` e `Kill` leva `DesmontarUnidade` a devolver `false` com mensagem que cita o motivo.
- [ ] Teste: um processo que termina com `Interrupt` leva `DesmontarUnidade` a devolver `true` e a sair do mapa.
- [ ] `rg -n "DesmontarUnidade\(nome\)" main.go` não encontra nada.
- [ ] No Windows: destrancar, gravar um arquivo de 1 MB na unidade, trancar. A letra some do Explorer e o arquivo aparece no provedor (conferir com `rclone ls <nome>:`).

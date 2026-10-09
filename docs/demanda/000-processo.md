# 000 — Processo de demandas

## Regra

1. **A demanda existe antes do código.** Nenhum PR de código abre sem um arquivo `docs/demanda/NNN-*.md` já na `master`. O PR cita o número no título, por exemplo `fix(001): ...`.
2. **Uma preocupação por demanda.** Se aparecer uma segunda, ela vira outra demanda com outro número.
3. **Cada demanda tem três seções obrigatórias:**
   - **Contexto:** o problema, com arquivo e função onde ele está no código de hoje.
   - **O que fica de fora:** o que esta demanda não faz, mesmo que pareça perto.
   - **Pronto quando:** itens que alguém de fora do projeto consegue conferir sozinho, com um comando, um teste ou um passo na tela. "Funciona" e "está melhor" não contam.
4. **Ordem:** pelo risco à estabilidade e aos segredos. Uma demanda só começa depois das que ela lista em "Depende de".
5. **Decisão duradoura vira ADR.** Se a demanda escolhe entre alternativas que vão durar, o ADR é escrito antes do código e fica em `docs/adr/`.
6. **Fechamento:** o PR que cumpre todos os itens de "Pronto quando" muda o estado da demanda para `Feita` e põe o hash do merge.

## Modelo

```markdown
# NNN — Título curto

- Estado: Aberta | Em andamento | Feita (<sha>)
- Risco: Alto | Médio | Baixo — estabilidade | segredos | dados | uso
- Onde: `arquivo.go:Funcao`
- Depende de: NNN, NNN

## Contexto

## O que muda

## O que fica de fora

## Pronto quando

- [ ] ...
```

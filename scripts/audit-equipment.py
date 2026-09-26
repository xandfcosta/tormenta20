#!/usr/bin/env python3
"""Confere o EQUIPAMENTO do app contra o livro, item por item (ALE-393).

Rodar é ATO DELIBERADO, como o `genoracle` e os auditores irmãos: a ferramenta
PROPÕE, o diff se revisa contra o PDF, e é a revisão que decide. Ela não escreve.

    python3 scripts/audit-equipment.py

As âncoras são o NOME e o PREÇO
--------------------------------
O capítulo 3 põe todo item numa TABELA — Armas (p144-145), Armaduras & Escudos
(p153), Itens Gerais (p156-157), Melhorias (p165) —, e toda tabela traz preço e
espaços, que é o par que o catálogo também afirma. O que divergir depois de o
nome casar é defeito do catálogo.

Item que não aparece em tabela nenhuma sai como NÃO MEDIDO, que não é a mesma
coisa que "correto".

A armadilha: a linha tem DOIS itens
------------------------------------
As tabelas de Itens Gerais são de duas colunas, e uma linha traz dois verbetes
com os campos de cada um lado a lado:

    ['Óleo', 'T$ 0,1', '0,5', 'Tabardo', 'T$ 10', '1']

Ler o primeiro nome da linha e o primeiro preço dela audita SÓ a coluna da
esquerda, e faz a direita sair como "não casou" — pareceu, na primeira medição,
que faltavam 15 itens no catálogo. O bloco de um item vai do NOME dele até o
próximo nome; assim a cobertura foi de 35 para 105.

A geometria do PDF é a mesma dos outros auditores, e mora no `t20pdf`.
"""
import argparse
import collections
import json
import re
import sys

from t20pdf import RAIZ, chave, linhas_com_coordenada

ITENS = str(RAIZ / 'engine-go/domain/catalog/data/items.json')
OFFSET = 6  # página do PDF = página do livro + 6

# As tabelas de ITEM do capítulo 3, com as páginas do LIVRO. Ficam de fora a
# 3-1 (dinheiro inicial), a 3-2 (dano por tamanho), a 3-4 (munições), a 3-7 e a
# 3-9 (tabelas de PREÇO de melhoria e material, que não são verbetes).
TABELAS = [
    ('Tabela 3-3', [144, 145], 'Armas'),
    ('Tabela 3-5', [153], 'Armaduras & Escudos'),
    ('Tabela 3-6', [156, 157], 'Itens Gerais'),
    ('Tabela 3-8', [165], 'Melhorias'),
]
RE_PRECO = re.compile(r'^T\$\s*([\d.,]+)$')


def catalogo() -> dict:
    with open(ITENS, encoding='utf-8') as f:
        bruto = json.load(f)
    itens = bruto if isinstance(bruto, list) else list(bruto.values())
    return {chave(i['name']): i for i in itens}


def preco_do_livro(texto: str):
    m = RE_PRECO.match(texto)
    if not m:
        return None
    return float(m.group(1).replace('.', '').replace(',', '.'))


def blocos_da_tabela(paginas, titulo: str, nomes: dict) -> list:
    """Um bloco por ITEM: do nome dele até o próximo nome da mesma linha."""
    saida = []
    for pagina in paginas:
        linhas = linhas_com_coordenada(pagina + OFFSET)
        cabecalho = [l for l in linhas if l[2].startswith(titulo)]
        if not cabecalho:
            continue
        y_do_titulo = cabecalho[0][1]
        por_y = collections.defaultdict(list)
        for x, y, t in linhas:
            if y > y_do_titulo:
                por_y[round(y)].append((x, t))
        for y in sorted(por_y):
            celulas = sorted(por_y[y])
            inicios = [i for i, (_x, t) in enumerate(celulas) if chave(t) in nomes]
            for n, i in enumerate(inicios):
                fim = inicios[n + 1] if n + 1 < len(inicios) else len(celulas)
                saida.append(celulas[i:fim])
    return saida


def main() -> None:
    argparse.ArgumentParser().parse_args()
    nomes = catalogo()
    vistos, conferidos, divergentes = set(), 0, []

    for titulo, paginas, nome_da_tabela in TABELAS:
        nesta = semPreco = 0
        for bloco in blocos_da_tabela(paginas, titulo, nomes):
            k = chave(bloco[0][1])
            item = nomes[k]
            vistos.add(k)
            precos = [p for p in (preco_do_livro(t) for _x, t in bloco) if p is not None]
            if not precos:
                semPreco += 1
                continue
            nesta += 1
            conferidos += 1
            if abs(precos[0] - item.get('price', 0)) > 0.001:
                divergentes.append((nome_da_tabela, item['name'], item.get('price'), precos[0]))
        print(f'  {nome_da_tabela:22s} preço conferido: {nesta:3d}   sem preço na tabela: {semPreco}')

    naoVistos = sorted(i['name'] for k, i in nomes.items() if k not in vistos)
    print(f'\nitens no catálogo:        {len(nomes)}')
    print(f'  com preço conferido:    {conferidos}')
    print(f'  NÃO MEDIDOS:            {len(naoVistos)}')
    print(f'  divergem do livro:      {len(divergentes)}')
    for tabela, nome, doCatalogo, doLivro in divergentes:
        print(f'    PREÇO  {tabela:20s} {nome[:28]:30s} catálogo={doCatalogo} livro={doLivro}')
    if naoVistos:
        print('\nNÃO MEDIDOS (não aparecem em tabela nenhuma do capítulo 3):')
        for nome in naoVistos:
            print(f'    {nome}')


if __name__ == '__main__':
    sys.exit(main())

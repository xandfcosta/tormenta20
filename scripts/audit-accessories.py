#!/usr/bin/env python3
"""Confere os ACESSÓRIOS do catálogo contra as Tabelas 8-13/14/15 (ALE-452).

São TRÊS tabelas e UMA lista de verbetes
-----------------------------------------
O livro separa os acessórios por magnitude — menores (p342), médios e maiores
(p343) —, mas os verbetes vêm numa lista só, em ORDEM ALFABÉTICA, atravessando
as páginas 342 a 349. Por isso aqui as três tabelas são lidas juntas: o que
procura verbete é o conjunto dos 65 nomes, e cada um volta sabendo de que tabela
veio (que é o que dá o preço e a faixa de `d%`).

A leitura das tabelas e a dos verbetes moram no `t20pdf` — este é o terceiro
auditor desta seção, e foi ele que as fez virar módulo.

O que se confere, e o que NÃO se confere
-----------------------------------------
Confere-se o que o livro imprime duas vezes ou declara: o nome existe na tabela
E tem verbete, o preço bate, a página citada é a do verbete, e a descrição do
catálogo cobre o texto do livro.

NÃO se confere a regra: diferente do item específico, um acessório não tem
composição em itálico — ele é um verbete solto, e o que ele faz é prosa. Quem
decide se aquilo virou modificador, gesto ou nota de `unmodeled` é o guarda do
catálogo, não este auditor.
"""
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import t20pdf as P

OFFSET_DO_PDF = 6
FOLGA = 3.0

# AS TRÊS TABELAS, com as coordenadas medidas na página. O x do `d%` é FAIXA
# porque a célula é alinhada à direita — a borda de baixo já custou a última
# linha da 8-13 ("99-100" começa 2 pontos à esquerda de "97-98").
TABELAS = (
    {'tabela': '8-13', 'nome': 'menores', 'pagina': 348,
     'x_do_dado': (293.0, 302.0), 'x_do_nome': 330.0, 'x_do_preco': (460.0, 470.0)},
    {'tabela': '8-14', 'nome': 'médios', 'pagina': 349,
     'x_do_dado': (76.0, 84.0), 'x_do_nome': 111.7, 'x_do_preco': (242.0, 250.0)},
    {'tabela': '8-15', 'nome': 'maiores', 'pagina': 349,
     'x_do_dado': (313.0, 322.0), 'x_do_nome': 348.7, 'x_do_preco': (480.0, 492.0)},
)

# OS VERBETES atravessam QUATRO páginas (342 a 345), e a margem MUDA de par
# para ímpar: a coluna esquerda começa em 51 na par e em 70,9 na ímpar.
#
# A lista acaba na p345, com a Vassoura voadora — e o que vem depois são os
# ARTEFATOS, cinco páginas de prosa que o livro tira da geração aleatória de
# propósito ("não há tabelas para geração aleatória de artefatos"). O título
# deles é CENTRALIZADO (x=221,9), fora das duas colunas, então nenhuma sentinela
# de texto o alcança: quem para a leitura é a última página da lista.
PAGINAS_DOS_VERBETES = tuple(range(348, 352))
COLUNA_PAR = ((51.0, 68.0), (289.1, 306.1))
COLUNA_IMPAR = ((70.9, 87.9), (309.0, 326.0))
COLUNAS = {pagina: (COLUNA_PAR if pagina % 2 == 0 else COLUNA_IMPAR)
           for pagina in PAGINAS_DOS_VERBETES}

# AS REGIÕES DE TABELA a pular, e a p343 tem DUAS — uma em cada coluna. É por
# isso que a região é uma lista: um retângulo só não cobre a página que traz a
# 8-14 à esquerda e a 8-15 à direita.
REGIOES = {
    348: ((290.0, 520.0, 50.0, 390.0),),
    349: ((70.0, 270.0, 50.0, 395.0), (310.0, 520.0, 50.0, 395.0)),
}

FIM_DOS_VERBETES = ()

PISO_DA_COBERTURA = 0.95
CATALOGO = P.RAIZ / 'engine-go/domain/catalog/data/items.json'


def linhas_das_tabelas():
    """(tabela, inicio, fim, nome, preço) das três, com a tilagem conferida."""
    for tabela in TABELAS:
        linhas = list(P.linhas_de_tabela(tabela['pagina'], tabela['x_do_dado'],
                                         tabela['x_do_nome'], tabela['x_do_preco'],
                                         FOLGA))
        buraco = P.faixas_que_nao_fecham(linhas)
        if buraco:
            raise SystemExit(f'A TABELA {tabela["tabela"]} NÃO FECHA: {buraco}\n'
                             'Nada do que viria abaixo valeria — a leitura está torta.')
        for inicio, fim, nome, preco in linhas:
            yield tabela, inicio, fim, nome, preco


def do_catalogo():
    itens = json.loads(CATALOGO.read_text())
    return {P.chave(i['name']): i for i in itens if i.get('category') == 'apparel'}


def audita() -> int:
    linhas = list(linhas_das_tabelas())
    corpos = P.verbetes_por_nome(PAGINAS_DOS_VERBETES, COLUNAS, REGIOES,
                                 [nome for _t, _i, _f, nome, _p in linhas],
                                 FIM_DOS_VERBETES, OFFSET_DO_PDF, FOLGA)
    por_nome = do_catalogo()
    falhas, transcritos = [], 0

    for tabela in TABELAS:
        print(f'\n══ TABELA {tabela["tabela"]} (acessórios {tabela["nome"]}) '
              + '═' * 20)
        print(f'{"d%":>7} {"acessório":34} {"preço":>8}  catálogo')
        for linha in linhas:
            if linha[0] is not tabela:
                continue
            _t, inicio, fim, nome, preco = linha
            faixa = f'{inicio}-{fim}' if fim != inicio else str(inicio)
            entrada = por_nome.get(P.chave(nome))
            print(f'{faixa:>7} {nome:34} {preco:>8}  '
                  f'{entrada["id"] if entrada else "— AUSENTE"}')
            if nome not in corpos:
                falhas.append(f'{nome}: está na tabela {tabela["tabela"]} e não tem verbete')
            if entrada:
                transcritos += 1
                falhas += confere(entrada, nome, preco, corpos.get(nome))

    print(f'\n{transcritos}/{len(linhas)} transcritos')
    for falha in falhas:
        print(f'  ✗ {falha}')
    return len(falhas)


def confere(entrada, nome, preco, verbete) -> list:
    falhas = []
    if entrada.get('price') != preco:
        falhas.append(f'{nome}: o catálogo diz T$ {entrada.get("price")} e a tabela '
                      f'diz T$ {preco}')
    if verbete is None:
        return falhas
    if entrada.get('bookPage') != verbete['pagina']:
        falhas.append(f'{nome}: o catálogo cita p{entrada.get("bookPage")} e o verbete '
                      f'está na p{verbete["pagina"]}')
    texto = P.junta(verbete['corpo'])
    razao, _comuns, _total = P.cobertura(texto, entrada.get('description', ''))
    if razao < PISO_DA_COBERTURA:
        falhas.append(f'{nome}: a descrição cobre {razao:.0%} do verbete '
                      f'(piso {PISO_DA_COBERTURA:.0%})')
    return falhas


if __name__ == '__main__':
    raise SystemExit(min(1, audita()))

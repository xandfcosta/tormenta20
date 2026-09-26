#!/usr/bin/env python3
"""Confere `expertises.json` contra o Capítulo 2 do livro (p115-123). Seção da ALE-391.

O que este auditor mede
-----------------------
Os quatro campos que o catálogo afirma por perícia: `attribute`, `soTreinada`,
`penalidadeDeArmadura` e `bookPage`. Não mede a prosa dos USOS da perícia —
"Amortecer Queda", "Equilíbrio" e companhia são dezenas de parágrafos que o
catálogo nem guarda.

DUAS leituras independentes, e elas têm de concordar
-----------------------------------------------------
O livro imprime a mesma verdade duas vezes, e é isso que faz deste o auditor
mais barato de validar da família:

1. **A Tabela 2-1** (p115, coluna da direita) — uma linha por perícia, com
   `Atributo-chave`, `Somente Treinada?` e `Penalidade de Armadura?` em três
   células lidas por Y.
2. **A linha de estado do VERBETE** — cada perícia abre com o nome e, à direita
   dele, `Des • Treinada • Armadura`. Ela vive nas nove páginas do capítulo, e
   é ela que dá o `bookPage`.

Comparar as duas ANTES de comparar com o catálogo é o controle: um instrumento
que passou a ler errado discorda do outro, e a discordância aparece com o nome
da perícia. Uma leitura só daria uma tabela plausível — que é o que a ALE-230
chama de "o número errado com cara de resultado".

Como o estado se liga ao título: por PALAVRA, e em três condições
------------------------------------------------------------------
A semente é a abreviação do atributo; as marcas vêm à direita dela na mesma
altura; e o título é a palavra de nome de perícia mais próxima **à esquerda, na
mesma COLUNA, na mesma altura** (o estado desce até ~30pt abaixo do título). As
três são necessárias, e cada uma foi paga:

- **Por palavra, não por linha.** Na p119 o elemento `<line>` traz `Furtividade
  Des • Armadura Iniciativa` — título, estado, e o título da perícia da coluna
  VIZINHA, na mesma altura. Lendo a linha inteira não há como separá-los.
- **A coluna é parte da regra.** Sem ela o estado de Religião liga em
  Percepção, que é treinada nenhuma, e o auditor inventa uma divergência.
- **A largura da coluna varia**, então a distância título→estado não serve para
  nada: na p115 são 111pt, na p116, 195pt.

As colunas se acham na PROSA. A Tabela 2-1 tem colunas internas próprias, e
contá-las põe o título do Adestramento — impresso logo abaixo dela, na mesma
coluna da página — numa coluna diferente da do estado dele.

Uso: `python3 scripts/audit-expertises.py`. Precisa do PDF do livro (veja t20pdf).
"""
import json
import sys
from collections import Counter
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from t20pdf import (  # noqa: E402
    RAIZ, chave, coluna_de, linhas_com_coordenada, palavras_da_pagina)

PRIMEIRA, ULTIMA = 121, 129  # PDF
OFFSET_DO_PDF = 6  # livro = PDF - 6
PAGINA_DA_TABELA = 121
TABELA_COMECA_EM_X = 300  # a Tabela 2-1 divide a p115 com o primeiro verbete
EXPERTISES = RAIZ / 'engine-go/domain/catalog/data/expertises.json'

ABREVIACOES = {
    'For': 'strength', 'Des': 'dexterity', 'Con': 'constitution',
    'Int': 'intelligence', 'Sab': 'wisdom', 'Car': 'charisma',
}
MARCAS = ('Treinada', 'Armadura')
MESMA_ALTURA = 4.0  # pt; a marca desce um fio em relação à abreviação
# O título nasce na mesma altura do estado; a folga para baixo cabe o título de
# seção da p115, que empurra o primeiro verbete.
FOLGA_ACIMA, FOLGA_ABAIXO = 30.0, 5.0
MINIMO_DE_LINHAS_POR_COLUNA = 5
RECUO_MAXIMO = 40.0  # pt; acima disso é outra coluna, abaixo é recuo de parágrafo


def inicios_das_colunas_por_linha(linhas) -> list[float]:
    """Os x onde as colunas começam, por frequência das LINHAS.

    O `inicios_das_colunas` do módulo conta BLOCOS e de propósito não funde
    faixas de x. Aqui se contam LINHAS, e aí aparece um x que não é coluna: o
    RECUO de primeira linha de parágrafo, 17pt à direita do começo da coluna e
    frequente o bastante para passar por uma. Fundir o que está a menos de
    `RECUO_MAXIMO` de um começo já aceito é o que separa recuo de coluna.
    """
    contagem = Counter(round(x, 1) for x, _y, _t in linhas)
    inicios: list[float] = []
    for x in sorted(x for x, n in contagem.items() if n >= MINIMO_DE_LINHAS_POR_COLUNA):
        if not inicios or x - inicios[-1] > RECUO_MAXIMO:
            inicios.append(x)
    return inicios or [min((x for x, _y, _t in linhas), default=0.0)]


def le_tabela(nomes: dict) -> dict:
    """A Tabela 2-1 → ({id: leitura}, as alturas que ela ocupa).

    Célula por célula, agrupada por Y.

    As três colunas são blocos DIFERENTES; o que diz que "Acrobacia", "Des",
    "—" e "sim" são a mesma linha é o y das quatro ser igual.
    """
    por_linha: dict[float, list[tuple[float, str]]] = {}
    for x, y, texto in linhas_com_coordenada(PAGINA_DA_TABELA):
        if x > TABELA_COMECA_EM_X:
            por_linha.setdefault(round(y, 1), []).append((x, texto))
    fora, alturas = {}, set()
    for altura, celulas in por_linha.items():
        celulas.sort()
        id_pericia = nomes.get(chave(celulas[0][1]))
        if id_pericia is None or len(celulas) != 4:
            continue
        _nome, atributo, treinada, armadura = (c for _x, c in celulas)
        fora[id_pericia] = {
            'attribute': ABREVIACOES.get(atributo),
            'soTreinada': treinada == 'sim',
            'penalidadeDeArmadura': armadura == 'sim',
        }
        alturas.add(altura)
    return fora, alturas


def le_verbetes(nomes: dict, alturas_da_tabela: set) -> tuple[dict, list[str]]:
    """A linha de estado de cada verbete → {id: leitura}, com a página do livro.

    Lida por PALAVRA: a abreviação do atributo é a semente, as marcas vêm à
    direita dela na mesma altura, e o título é a palavra de nome de perícia
    mais próxima À ESQUERDA, na MESMA COLUNA. As três condições são
    necessárias — sem a coluna, o estado de Religião liga em Percepção, que é
    treinada nenhuma, e o auditor devolve uma divergência inventada.
    """
    fora, queixas = {}, []
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        # As colunas se acham na PROSA. A Tabela 2-1 tem colunas internas
        # próprias (atributo, treinada, armadura), e contá-las faz o título do
        # Adestramento — que é impresso logo abaixo da tabela, na mesma coluna
        # da página — cair numa coluna diferente da do estado dele.
        prosa = [r for r in linhas_com_coordenada(pagina)
                 if not any(abs(r[1] - altura) < MESMA_ALTURA
                            for altura in alturas_da_tabela)]
        inicios = inicios_das_colunas_por_linha(prosa)
        palavras = palavras_da_pagina(pagina)
        for x, y, texto in palavras:
            if texto not in ABREVIACOES:
                continue
            if pagina == PAGINA_DA_TABELA and any(
                    abs(y - altura) < MESMA_ALTURA for altura in alturas_da_tabela):
                continue  # é célula da Tabela 2-1, já lida pelo outro instrumento
            marcas = {p for px, py, p in palavras
                      if p in MARCAS and px > x and abs(py - y) < MESMA_ALTURA}
            coluna = coluna_de(inicios, x)
            candidatos = [(px, py, p) for px, py, p in palavras
                          if chave(p) in nomes and px < x
                          and coluna_de(inicios, px) == coluna
                          and y - FOLGA_ACIMA <= py <= y + FOLGA_ABAIXO]
            if not candidatos:
                queixas.append(f'estado {texto!r} na p{pagina - OFFSET_DO_PDF} '
                               f'sem título de perícia à esquerda na mesma coluna')
                continue
            titulo = min(candidatos, key=lambda c: abs(c[1] - y))[2]
            fora[nomes[chave(titulo)]] = {
                'attribute': ABREVIACOES[texto],
                'soTreinada': 'Treinada' in marcas,
                'penalidadeDeArmadura': 'Armadura' in marcas,
                'bookPage': pagina - OFFSET_DO_PDF,
            }
    return fora, queixas


CAMPOS = ('attribute', 'soTreinada', 'penalidadeDeArmadura')


def main() -> int:
    catalogo = json.load(open(EXPERTISES, encoding='utf-8'))
    nomes = {chave(p['name']): p['id'] for p in catalogo}
    tabela, alturas = le_tabela(nomes)
    verbetes, queixas = le_verbetes(nomes, alturas)

    # O CONTROLE: os dois instrumentos antes do catálogo. Um que passou a ler
    # errado discorda do outro, e a discordância sai com o nome da perícia.
    brigas = 0
    for id_pericia in sorted(set(tabela) & set(verbetes)):
        for campo in CAMPOS:
            if tabela[id_pericia][campo] != verbetes[id_pericia][campo]:
                print(f'  INSTRUMENTOS DISCORDAM {id_pericia}.{campo}: '
                      f'tabela {tabela[id_pericia][campo]} × verbete '
                      f'{verbetes[id_pericia][campo]}')
                brigas += 1

    divergem = 0
    for pericia in catalogo:
        lido = verbetes.get(pericia['id'])
        if lido is None:
            queixas.append(f'{pericia["name"]}: sem linha de estado no capítulo')
            continue
        for campo in CAMPOS + ('bookPage',):
            if pericia[campo] != lido[campo]:
                print(f'  {pericia["name"]}: {campo}: catálogo {pericia[campo]} '
                      f'× livro {lido[campo]}')
                divergem += 1
        if pericia['id'] not in tabela:
            queixas.append(f'{pericia["name"]}: sem linha na Tabela 2-1')

    for queixa in queixas:
        print(f'  NÃO MEDIDO {queixa}')

    print(f'\nperícias: {len(catalogo)} | lidas na Tabela 2-1: {len(tabela)} '
          f'| lidas no verbete: {len(verbetes)} | instrumentos discordam: {brigas} '
          f'| divergem do livro: {divergem}')
    return 1 if divergem or brigas or queixas else 0


if __name__ == '__main__':
    raise SystemExit(main())

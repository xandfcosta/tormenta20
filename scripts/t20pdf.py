#!/usr/bin/env python3
"""A leitura do PDF do livro por COORDENADA, para os auditores de catálogo.

Ela nasceu no `audit-bestiary.py` (ALE-151), foi copiada para o
`audit-spells.py` (ALE-340), e virou módulo quando o terceiro auditor ia copiá-la
de novo (ALE-391). As duas cópias já tinham divergido — em docstring, e uma
delas descrevia uma tupla de três valores onde o código devolve quatro.

O que ela resolve, e por que não serve `pdftotext -layout`
-----------------------------------------------------------
O `-layout` junta colunas VIZINHAS na mesma linha de texto: a linha de atributos
de uma criatura aparece colada à criatura errada, e isso já quase fez corrigir a
Hidra AO CONTRÁRIO. A leitura aqui é por coordenada.

E a geometria VARIA: há página de duas e de três colunas, e o título de seção é
centralizado e atravessa a calha — fundir faixas de x junta as duas colunas numa
só. As colunas se acham por FREQUÊNCIA do x onde o corpo começa.

Precisa do `pdftotext` (poppler) e do PDF do livro.
"""
import html
import os
import pathlib
import re
import subprocess
import unicodedata
from collections import Counter

# O CATÁLOGO sai da localização do script; o LIVRO, não — e a diferença é que o
# PDF é gitignorado (ele não é nosso para distribuir). Numa worktree, o
# catálogo a auditar é o de lá e o livro continua no checkout principal.
#
# Com um caminho fixo para os DOIS, rodar numa worktree audita o catálogo do
# checkout principal: o relatório sai sobre um arquivo que não é o que se está
# editando, e as correções parecem não ter pegado. Custou uma rodada.
RAIZ = pathlib.Path(__file__).resolve().parent.parent
PDF = os.environ.get('T20_BOOK_PDF') or str(RAIZ / 't20-book.pdf')
if not pathlib.Path(PDF).exists():
    # O checkout principal é o palpite seguinte, e ele é DITO: um auditor que
    # caísse em silêncio num PDF vazio reportaria 198 "não medidas" com cara de
    # resultado.
    vizinho = pathlib.Path('/mnt/HD/projects/tormenta20/t20-book.pdf')
    if not vizinho.exists():
        raise SystemExit(
            f'não achei o livro em {PDF}. Ele é gitignorado — aponte o '
            f'T20_BOOK_PDF para o PDF do checkout principal.')
    PDF = str(vizinho)
RE_BLOCO = re.compile(
    r'<block xMin="([\d.]+)" yMin="([\d.]+)" xMax="([\d.]+)"[^>]*>(.*?)</block>', re.S)
RE_LINHA = re.compile(r'<line[^>]*>(.*?)</line>', re.S)
RE_PALAVRA = re.compile(
    r'<word xMin="([\d.]+)" yMin="([\d.]+)"[^>]*>(.*?)</word>', re.S)



def blocos_da_pagina(pagina: int):
    """(xMin, xMax, yMin, [linhas]) de cada bloco declarado pelo PDF."""
    xml = subprocess.run(
        ['pdftotext', '-bbox-layout', '-f', str(pagina), '-l', str(pagina), PDF, '-'],
        capture_output=True, text=True).stdout
    for m in RE_BLOCO.finditer(xml):
        x0, ybloco, x1 = float(m.group(1)), float(m.group(2)), float(m.group(3))
        linhas, y0 = [], None
        for lm in RE_LINHA.finditer(m.group(4)):
            palavras = [
                (float(x), float(y), html.unescape(t))
                for x, y, t in RE_PALAVRA.findall(lm.group(1))
            ]
            if not palavras:
                continue
            if y0 is None:
                y0 = palavras[0][1]
            palavras.sort(key=lambda w: w[0])
            linhas.append(' '.join(t for _x, _y, t in palavras))
        if linhas:
            yield x0, x1, y0 if y0 is not None else ybloco, linhas

MINIMO_DE_BLOCOS_POR_COLUNA = 3


def inicios_das_colunas(bs) -> list[float]:
    """Os x onde as colunas COMEÇAM, achados por frequência.

    Não serve fundir faixas de x: o título de seção é centralizado e atravessa
    a calha, e uma única linha dessas funde as duas colunas numa só — foi assim
    que o Orc e o Glop saíram interfoliados. O que é estável é o x onde o corpo
    começa: numa página de duas colunas ele aparece dezenas de vezes em dois
    valores, e tudo que começa mais à direita (o "ND 1/4" alinhado à direita, a
    linha de atributos centralizada) pertence à coluna que vem ANTES dele.
    """
    contagem = Counter(round(b[0], 1) for b in bs)
    inicios = sorted(x for x, n in contagem.items() if n >= MINIMO_DE_BLOCOS_POR_COLUNA)
    return inicios or [min((b[0] for b in bs), default=0.0)]


def coluna_de(inicios: list[float], x0: float) -> int:
    """A qual coluna pertence um bloco que começa em `x0`.

    Tudo que começa mais à DIREITA de um início — o "ND 1/4" alinhado à direita,
    a linha de atributos centralizada — pertence à coluna que vem ANTES dele.
    """
    cabem = [i for i, ini in enumerate(inicios) if x0 >= ini - 2]
    return cabem[-1] if cabem else 0

def linhas_da_pagina(pagina: int) -> list[str]:
    """As linhas da página na ordem de LEITURA: coluna por coluna, de cima para
    baixo."""
    bs = list(blocos_da_pagina(pagina))
    if not bs:
        return []
    inicios = inicios_das_colunas(bs)

    bs.sort(key=lambda b: (coluna_de(inicios, b[0]), b[2]))
    saida: list[str] = []
    for _x0, _x1, _y, linhas in bs:
        saida.extend(linhas)
    return saida
MENOS = '–−—'  # o livro usa travessão, não hífen, nos negativos


def num(t: str) -> int:
    """"+2", "–1" e "−1" viram int. O livro escreve o negativo com TRAVESSÃO."""
    return int(t.strip().translate({ord(c): '-' for c in MENOS}).replace('+', ''))


RE_HIFEN = re.compile(r'(\w)-\s+(\w)')


def junta(corpo) -> str:
    """Junta linhas DESFAZENDO a hifenização de quebra.

    O livro quebra palavra no fim da linha ("subter-" / "râneos"), e juntar sem
    cuidado produz "subter- râneos" no meio de uma frase — o que faz um `find`
    pelo nome de uma habilidade não achar a habilidade que está lá.
    """
    return RE_HIFEN.sub(r'\1\2', ' '.join(corpo))


def chave(nome: str) -> str:
    """Normaliza para casar nome do livro com nome do catálogo."""
    sem_acento = ''.join(
        c for c in unicodedata.normalize('NFD', nome.lower())
        if unicodedata.category(c) != 'Mn')
    return re.sub(r'[^a-z0-9]+', '', sem_acento)
def normaliza_frase(t: str) -> str:
    """O texto de um aprimoramento, comparável: sem acento, sem pontuação, sem
    espaço duplicado. A comparação é por CONTEÚDO — o catálogo reescreve as
    frases do livro em forma mais curta de propósito."""
    sem_acento = ''.join(
        c for c in unicodedata.normalize('NFD', t.lower())
        if unicodedata.category(c) != 'Mn')
    return re.sub(r'\s+', ' ', re.sub(r'[^a-z0-9 ]+', ' ', sem_acento)).strip()

def linhas_com_coordenada(pagina: int) -> list[tuple[float, float, str]]:
    """(x, y, texto) de cada LINHA da página, sem agrupar por bloco.

    O `linhas_da_pagina` colapsa as linhas de um bloco numa lista e perde o y de
    cada uma — o que serve para ler PROSA em ordem de leitura, e não serve para
    ler TABELA.

    Numa tabela, a coluna do nível e a da habilidade são blocos DIFERENTES, e o
    que diz que "6º" e "Fúria +3" são a mesma LINHA é o y das duas ser igual.
    Parear por índice erra na primeira habilidade que quebra em duas linhas —
    e a do Bárbaro quebra (ALE-392).
    """
    xml = subprocess.run(
        ['pdftotext', '-bbox-layout', '-f', str(pagina), '-l', str(pagina), PDF, '-'],
        capture_output=True, text=True).stdout
    saida = []
    for m in re.finditer(r'<line xMin="([\d.]+)" yMin="([\d.]+)"[^>]*>(.*?)</line>', xml, re.S):
        texto = re.sub(r'\s+', ' ', html.unescape(re.sub(r'<[^>]+>', ' ', m.group(3)))).strip()
        if texto:
            saida.append((float(m.group(1)), float(m.group(2)), texto))
    return saida

#!/usr/bin/env python3
"""Confere as ESTATÍSTICAS DE OBJETOS contra o livro (Tab. 5-4, p239). ALE-423.

O que este auditor mede, e por que a tabela não é uma transcrição
-----------------------------------------------------------------
O livro imprime quatro colunas — Tamanho, Def, RD e PV — e o motor guarda DUAS
ESCADAS e uma coluna. A Defesa é função do tamanho, e a página diz isso com
essas palavras: *"faça um ataque contra a Defesa do objeto, definida por sua
categoria de tamanho"*. A RD é função do material, que a página afirma sem
tabular. Sobra o PV, que é o único número por verbete.

Então o que se confere aqui são as 44 células da primeira metade da tabela
contra o que o `domain/engine/objects.go` RECONSTRÓI delas. Um degrau de escada
editado por engano para de reproduzir a página, e o auditor nomeia a linha.

O que este auditor NÃO pode medir, e vale dizer
------------------------------------------------
**O material de cada verbete não está impresso.** O livro não escreve "Barril —
madeira": ele escreve RD 5, e "madeira" é LEITURA, feita na ALE-423 a partir das
11 linhas. O que se mede, portanto, é o par (leitura, escada) reproduzindo a RD
impressa — o que pega um degrau errado e um material trocado, e não pega os dois
errados na mesma direção. As outras três colunas são conferência cheia.

A SEGUNDA METADE da tabela (armas, armaduras e escudos) está fora de propósito:
ela é o alvo da manobra quebrar e ainda não tem casa no motor. Quando tiver, o
`PRIMEIRA_METADE` abaixo cresce e este auditor mede as 21.

Lido por COORDENADA e não por `-layout`
----------------------------------------
A armadilha é a mesma que o `audit-bestiary.py` documenta: o `pdftotext -layout`
junta colunas VIZINHAS na mesma linha de texto, e a p239 tem duas tabelas lado a
lado — a 5-3 à esquerda e a 5-4 à direita. Uma linha de modificador de ataque
apareceria colada a um objeto. Cada palavra vai para a coluna de centro mais
próximo, e o centro vem do CABEÇALHO medido na própria página.

Uso: `python3 scripts/audit-objects.py`. Precisa do PDF do livro (t20pdf).
"""
import collections
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from t20pdf import RAIZ, palavras_da_pagina  # noqa: E402

PAGINA = 245  # PDF; livro = PDF − 6, então esta é a p239
REGRA = RAIZ / 'engine-go/domain/engine/objects.go'

# A FAIXA da primeira metade, em y: do "Pergaminho" ao "Celeiro". Ela exclui de
# propósito o cabeçalho (y≈96), o subtítulo "Objetos Gerais" (y≈119) e a segunda
# metade (y≈300 para baixo) — medido na página.
PRIMEIRA_METADE = (130.0, 295.0)
# A coluna da DIREITA começa aqui. À esquerda está a Tabela 5-3, que é de outro
# auditor e que uma leitura por layout traria para dentro desta.
X_MINIMO = 310.0
# Os cabeçalhos que dão o centro de cada coluna, na ordem da tabela.
CABECALHOS = ('Exemplo', 'Tamanho', 'Def', 'RD', 'PV')
# Duas palavras na mesma linha quando o y difere menos que isto.
MESMA_LINHA = 2.0


def centros_das_colunas() -> dict:
    """{nome do cabeçalho: x}, lido da própria página.

    Falha quando não acha os cinco: um centro chutado espalharia as células pelas
    colunas erradas e produziria uma lista de divergências com cara de
    descoberta (ALE-294).
    """
    achados = {}
    for x, y, texto in palavras_da_pagina(PAGINA):
        if 90.0 <= y <= 105.0 and x >= X_MINIMO and texto in CABECALHOS:
            achados[texto] = x
    faltam = [c for c in CABECALHOS if c not in achados]
    if faltam:
        raise SystemExit(
            f'não achei o cabeçalho {faltam} da Tab. 5-4 na p{PAGINA} do PDF. '
            f'Sem o centro de cada coluna este auditor não mede nada — e mediria '
            f'errado em silêncio se chutasse.')
    return achados


def linhas_impressas() -> list:
    """As 11 linhas da primeira metade: [(nome, tamanho, def, rd, pv)]."""
    centros = centros_das_colunas()
    baixo, alto = PRIMEIRA_METADE
    por_linha = collections.defaultdict(lambda: collections.defaultdict(list))
    for x, y, texto in palavras_da_pagina(PAGINA):
        if not (baixo <= y <= alto and x >= X_MINIMO):
            continue
        coluna = min(CABECALHOS, key=lambda c: abs(x - centros[c]))
        chave = next((k for k in por_linha if abs(k - y) < MESMA_LINHA), y)
        por_linha[chave][coluna].append((x, texto))

    fora = []
    for y in sorted(por_linha):
        celulas = {c: ' '.join(t for _x, t in sorted(ws))
                   for c, ws in por_linha[y].items()}
        faltam = [c for c in CABECALHOS if c not in celulas]
        if faltam:
            raise SystemExit(
                f'a linha em y={y:.1f} da Tab. 5-4 veio sem {faltam}: {celulas}. '
                f'Uma linha lida pela metade reprovaria o motor por um defeito da '
                f'LEITURA — este auditor para em vez de acusar.')
        numeros = {}
        for coluna in ('Def', 'RD', 'PV'):
            if not re.fullmatch(r'\d+', celulas[coluna]):
                raise SystemExit(
                    f'a célula {coluna} de {celulas["Exemplo"]!r} veio {celulas[coluna]!r}, '
                    f'que não é número — a leitura da página mudou de forma.')
            numeros[coluna] = int(celulas[coluna])
        fora.append((celulas['Exemplo'], celulas['Tamanho'],
                     numeros['Def'], numeros['RD'], numeros['PV']))
    return fora


def escada(nome: str) -> dict:
    """{chave: número} de um `map[string]int` do arquivo da regra."""
    fonte = REGRA.read_text(encoding='utf-8')
    abre = fonte.index('{', fonte.index(f'var {nome} = map[string]int'))
    corpo = fonte[abre + 1:fonte.index('}', abre)]
    degraus = {m.group(1): int(m.group(2))
               for m in re.finditer(r'"([^"]+)":\s*(-?\d+)', corpo)}
    if not degraus:
        raise SystemExit(f'não consegui ler nenhum degrau de `{nome}` em {REGRA.name}')
    return degraus


def exemplos_do_motor() -> dict:
    """{nome: (tamanho, material, pv)} do `objectExamplesOfTheBook`."""
    fonte = REGRA.read_text(encoding='utf-8')
    abre = fonte.index('{', fonte.index('var objectExamplesOfTheBook = []ObjectExample'))
    corpo = fonte[abre:fonte.index('\n}', abre)]
    linha = re.compile(
        r'Name:\s*"([^"]+)",\s*Size:\s*"([^"]+)",\s*Material:\s*"([^"]+)",\s*'
        r'HitPoints:\s*(\d+)')
    fora = {m.group(1): (m.group(2), m.group(3), int(m.group(4)))
            for m in linha.finditer(corpo)}
    if not fora:
        raise SystemExit(
            f'não consegui ler nenhum exemplo de `objectExamplesOfTheBook` em '
            f'{REGRA.name} — a forma da lista mudou e este auditor mediria zero '
            f'verbetes com cara de "tudo certo".')
    return fora


def sem_acento(t: str) -> str:
    """A chave normalizada, como o `normalizeSizeKey` do motor a escreve."""
    for de, para in (('é', 'e'), ('ê', 'e'), ('í', 'i'), ('ú', 'u'), ('ç', 'c'),
                     ('ã', 'a'), ('â', 'a'), ('ó', 'o'), ('õ', 'o'), ('á', 'a')):
        t = t.replace(de, para)
    return t.lower()


def main() -> int:
    impressas = linhas_impressas()
    defesa_por_tamanho = escada('defenseByObjectSize')
    rd_por_material = escada('damageReductionByObjectMaterial')
    exemplos = exemplos_do_motor()

    divergem, celulas = 0, 0
    for nome, tamanho, defesa, rd, pv in impressas:
        if nome not in exemplos:
            print(f'  {nome}: está impresso na Tab. 5-4 e não está no motor')
            divergem += 1
            continue
        do_motor_tamanho, material, do_motor_pv = exemplos[nome]

        celulas += 1
        if sem_acento(do_motor_tamanho) != sem_acento(tamanho):
            print(f'  {nome}: o livro diz {tamanho!r} e o motor diz {do_motor_tamanho!r}')
            divergem += 1

        celulas += 1
        da_escada = defesa_por_tamanho.get(sem_acento(tamanho))
        if da_escada != defesa:
            print(f'  {nome} ({tamanho}): a escada do tamanho dá Defesa {da_escada} '
                  f'e o livro imprime {defesa}')
            divergem += 1

        celulas += 1
        da_escada = rd_por_material.get(sem_acento(material))
        if da_escada != rd:
            print(f'  {nome} ({material}): a escada do material dá RD {da_escada} '
                  f'e o livro imprime {rd}')
            divergem += 1

        celulas += 1
        if do_motor_pv != pv:
            print(f'  {nome}: o motor dá {do_motor_pv} PV e o livro imprime {pv}')
            divergem += 1

    sobrando = sorted(set(exemplos) - {n for n, *_ in impressas})
    for nome in sobrando:
        print(f'  {nome}: está no motor e não achei na Tab. 5-4 impressa')
        divergem += 1

    print(f'\nlinhas impressas lidas da p239: {len(impressas)} de 11 | '
          f'células conferidas: {celulas} | divergem: {divergem}')
    print('o MATERIAL de cada verbete não está impresso no livro — ele é leitura '
          '(ALE-423), e o que se mede é (leitura + escada) reproduzindo a RD')
    return 1 if divergem or len(impressas) != 11 else 0


if __name__ == '__main__':
    raise SystemExit(main())

#!/usr/bin/env python3
"""Confere os ARTEFATOS do catálogo contra as p346-349 (ALE-455).

O denominador é INVERTIDO, e aqui ele não tinha escolha
--------------------------------------------------------
Os outros auditores desta família tilam uma tabela de `d%`: a tabela diz quantos
verbetes existem, e a soma das faixas de 1 a 100 denuncia a leitura torta antes
de encostar no catálogo. O artefato não tem tabela — a p346 é explícita: *"não há
tabelas para geração aleatória de artefatos, nem custo para fabricá-los"*. É a
ÚNICA família do Capítulo 8 sem o denominador de graça.

Então ele vem do TIPO. O cabeçalho de seção do Capítulo 8 é desenhado a 22,281
pontos de altura e o corpo a 16,2 — "Encantos" (p335), "Armas Específicas"
(p336), "Poções" (p341) e os cinco artefatos são todos a mesma altura. Dentro das
páginas 346-349 não há outra seção, então os cabeçalhos dali SÃO a lista de
artefatos, medida na página em vez de escrita à mão.

Isso é o que o auditor existe para pegar: eu contei TRÊS artefatos lendo as
p346-347, e são CINCO. Uma lista escrita à mão teria concordado comigo.

O CONTROLE vem antes do resultado
----------------------------------
Uma lista de reprovados vazia e um medidor que não casa com nada têm a mesma cor
no terminal. Antes de medir a seção, a mesma varredura roda na p335 e na p341,
onde se sabe o que ela tem de achar — se o cabeçalho de "Encantos" não aparecer,
o medidor está cego e o auditor para, em vez de anunciar "nenhum artefato
faltando".

O que se confere
-----------------
De cada artefato: ele tem verbete no catálogo (pelo nome), e a página citada é a
página do cabeçalho dele. Do Baralho, que é o único com tabela própria: as 22
cartas do app são as 22 da p347, pela FAIXA de `d%` e pelo nome, e a faixa tila
1 a 100 — que é o denominador de graça, aqui aplicado à tabela de dentro do
verbete.

NÃO se confere a regra. Quem decide se o que o verbete faz virou modificador,
gesto ou nota de `unmodeled` são os guardas do catálogo. O que este auditor faz
pela nota é IMPRIMI-LA, toda corrida, porque "a nota envelheceu" não se mede por
varredura de texto — se confere no olho, e o relatório é onde o olho passa.
"""
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import t20pdf as P

OFFSET_DO_PDF = 6

# A ALTURA do cabeçalho de seção do Capítulo 8, medida. A folga é apertada de
# propósito: a legenda da tabela do Baralho, na mesma página, é 20,604 — e uma
# janela larga leria "O Baralho do Caos" (a legenda) e "o Baralho do caos" (o
# cabeçalho) como dois artefatos.
ALTURA_DO_CABECALHO = 22.281
FOLGA_DA_ALTURA = 0.1

# A SEÇÃO, em páginas do livro. Ela começa onde a lista de acessórios acaba e
# termina na última página do capítulo.
PAGINAS_DA_SECAO = (346, 347, 348, 349)

# AS DUAS PÁGINAS DE CONTROLE, e o que a varredura TEM de achar nelas. Elas são
# de fora da seção de propósito: provam que a altura reconhece cabeçalho de
# seção em geral, e não só o que está nas quatro páginas medidas.
CONTROLE = {335: 'Encantos', 341: 'Poções'}

# A TABELA DO BARALHO, na p347. O `d%` é faixa porque "99-100" começa dois
# pontos à esquerda das outras, e não há coluna de preço — o terceiro campo da
# tabela é o resumo do efeito, que o auditor não compara (o catálogo guarda o
# parágrafo inteiro, não o resumo).
TABELA_DO_BARALHO = {'pagina': 353, 'x_do_dado': (310.0, 316.0), 'x_do_nome': 343.0}
SEM_COLUNA_DE_PRECO = (900.0, 910.0)
FOLGA = 3.0

ITENS = P.RAIZ / 'engine-go/domain/catalog/data/items.json'
TABELAS_DO_MESTRE = P.RAIZ / 'engine-go/domain/catalog/data/gm-tables.json'

RE_PALAVRA_COM_ALTURA = re.compile(
    r'<word xMin="([\d.]+)" yMin="([\d.]+)" xMax="[\d.]+" yMax="([\d.]+)">(.*?)</word>')


def cabecalhos_da_pagina(pdf_pagina: int) -> list[str]:
    """Os cabeçalhos de seção de UMA página, achados pela ALTURA do tipo.

    Agrupa por `y` porque um cabeçalho de quatro palavras são quatro `<word>` na
    mesma altura, e devolver quatro diria quatro seções.
    """
    import subprocess
    xml = subprocess.run(
        ['pdftotext', '-bbox', '-f', str(pdf_pagina), '-l', str(pdf_pagina), P.PDF, '-'],
        capture_output=True, text=True).stdout
    por_altura = {}
    for x, y0, y1, texto in RE_PALAVRA_COM_ALTURA.findall(xml):
        if abs((float(y1) - float(y0)) - ALTURA_DO_CABECALHO) > FOLGA_DA_ALTURA:
            continue
        por_altura.setdefault(round(float(y0), 0), []).append((float(x), texto))
    return [' '.join(t for _x, t in sorted(linha))
            for _y, linha in sorted(por_altura.items())]


def prova_que_o_medidor_ve():
    """O controle: a altura acha cabeçalho conhecido FORA da seção."""
    for pagina, esperado in CONTROLE.items():
        achados = cabecalhos_da_pagina(pagina + OFFSET_DO_PDF)
        if not any(P.chave(esperado) in P.chave(a) for a in achados):
            raise SystemExit(
                f'O MEDIDOR ESTÁ CEGO: a p{pagina} tem o cabeçalho {esperado!r} e a '
                f'varredura por altura {ALTURA_DO_CABECALHO} achou {achados}.\n'
                'Sem isto, "nenhum artefato faltando" e "não medi nada" são a mesma '
                'frase — o auditor para aqui de propósito.')


RE_FAIXA_DO_BARALHO = re.compile(r'^(\d{1,3})-(\d{1,3})$')


def tabela_do_baralho() -> list:
    """(inicio, fim, carta, None) das 22 linhas, lidas PALAVRA por palavra.

    Palavra e não linha, e não é zelo: o `linhas_de_tabela` da casa lê a célula
    do nome pelo x da LINHA, e numa das 22 ele erra. "Tentáculo" é o nome mais
    longo da coluna e encosta na coluna do efeito, então o `pdftotext` emite as
    duas células como UMA `<line>` e a carta sai "Tentáculo –1 em um atributo
    aleatório". As outras 21 passam, que é o pior jeito de errar — a tabela fica
    plausível e uma linha mente.

    É para isto que o `palavras_da_pagina` existe: a coluna se decide pelo x de
    cada palavra, e aí não há o que juntar. A quarta posição da tupla é o preço
    que esta tabela não tem, e está aí porque o `faixas_que_nao_fecham` — o
    denominador de graça — lê linhas de tabela de saque.
    """
    faixas, nomes = {}, {}
    for x, y, texto in P.palavras_da_pagina(TABELA_DO_BARALHO['pagina']):
        altura = round(y, 0)
        if TABELA_DO_BARALHO['x_do_dado'][0] <= x <= TABELA_DO_BARALHO['x_do_dado'][1]:
            casa = RE_FAIXA_DO_BARALHO.match(texto)
            if casa:
                faixas[altura] = (int(casa.group(1)), int(casa.group(2)))
        elif abs(x - TABELA_DO_BARALHO['x_do_nome']) < FOLGA:
            nomes[altura] = texto
    return [(inicio, fim, nomes.get(altura), None)
            for altura, (inicio, fim) in sorted(faixas.items())]


def artefatos_do_livro() -> dict:
    """{nome do cabeçalho: página do livro}, medido na seção."""
    achados = {}
    for pagina in PAGINAS_DA_SECAO:
        for titulo in cabecalhos_da_pagina(pagina + OFFSET_DO_PDF):
            achados[titulo] = pagina
    return achados


def do_catalogo() -> dict:
    itens = json.loads(ITENS.read_text())
    return {P.chave(i['name']): i for i in itens if i.get('artifact')}


def cartas_do_app() -> list:
    return json.loads(TABELAS_DO_MESTRE.read_text())['chaosDeck']


# O ARTIGO não é parte do nome: o livro escreve "A Espada-Deus", "os Rubis da
# Virtude".
ARTIGOS = ('a ', 'o ', 'as ', 'os ')

# E o cabeçalho da seção não é o nome do item em UM caso, declarado aqui para o
# casamento poder ser EXATO.
#
# A seção dos Rubis é plural porque são vinte gemas, e o catálogo guarda UMA: a
# p349 soma o efeito de cada uma — "cumulativos com efeitos já existentes —
# incluindo outros rubis" —, então o verbete é a gema e o jogador carrega
# quantas achar. Casar por prefixo em vez de declarar isto resolveria o caso e
# abriria outro: "Rubi" casa com "Rubis", e também com qualquer verbete futuro
# que comece com as mesmas letras.
NOME_DO_ITEM = {'Rubis da Virtude': 'Rubi da Virtude'}


def nome_do_item(titulo: str) -> str:
    baixo = titulo.lower()
    for artigo in ARTIGOS:
        if baixo.startswith(artigo):
            titulo = titulo[len(artigo):]
            break
    return NOME_DO_ITEM.get(titulo, titulo)


def audita() -> int:
    prova_que_o_medidor_ve()

    do_livro = artefatos_do_livro()
    no_app = do_catalogo()
    falhas = []

    print(f'══ ARTEFATOS (p{PAGINAS_DA_SECAO[0]}-{PAGINAS_DA_SECAO[-1]}) ' + '═' * 28)
    print(f'{"pág":>4} {"cabeçalho do livro":30} catálogo')
    casados = 0
    for titulo, pagina in sorted(do_livro.items(), key=lambda kv: (kv[1], kv[0])):
        entrada = no_app.get(P.chave(nome_do_item(titulo)))
        print(f'{pagina:>4} {titulo:30} {entrada["id"] if entrada else "— AUSENTE"}')
        if entrada is None:
            falhas.append(f'{titulo}: a p{pagina} traz este artefato e o catálogo não o tem')
            continue
        casados += 1
        if entrada.get('bookPage') != pagina:
            falhas.append(f'{titulo}: o catálogo cita p{entrada.get("bookPage")} e o '
                          f'cabeçalho está na p{pagina}')

    # E O LADO DE LÁ: artefato no app que o livro não imprime. Sem esta metade,
    # um verbete inventado passaria — a varredura só pergunta se o que o livro
    # tem está no app.
    do_livro_por_chave = {P.chave(nome_do_item(t)) for t in do_livro}
    for chave, entrada in sorted(no_app.items()):
        if chave not in do_livro_por_chave:
            falhas.append(f'{entrada["name"]}: está marcado como artefato no catálogo e '
                          f'nenhum cabeçalho das p{PAGINAS_DA_SECAO[0]}-'
                          f'{PAGINAS_DA_SECAO[-1]} corresponde a ele')

    # O CONTADOR é dos que CASARAM, e não dos dois tamanhos. "5 no livro, 5 no
    # app" saiu como "5/5" com um AUSENTE na tabela acima — instrumento mentindo
    # com cara de resultado, na primeira corrida deste auditor.
    print(f'\n{casados}/{len(do_livro)} transcritos')
    falhas += audita_o_baralho()
    imprime_as_notas(no_app)

    for falha in falhas:
        print(f'  ✗ {falha}')
    return len(falhas)


def audita_o_baralho() -> list:
    """As 22 cartas, pela faixa de `d%` e pelo nome."""
    linhas = tabela_do_baralho()
    buraco = P.faixas_que_nao_fecham(linhas)
    if buraco:
        raise SystemExit(f'A TABELA DO BARALHO NÃO FECHA NO LIVRO: {buraco}\n'
                         'Nada do que viria abaixo valeria — a leitura está torta.')

    cartas = cartas_do_app()
    falhas = []
    print(f'\n══ O BARALHO DO CAOS (p347) ' + '═' * 32)
    print(f'{"d%":>7} {"carta no livro":16} app')
    for inicio, fim, nome, _preco in linhas:
        faixa = f'{inicio}-{fim}' if fim != inicio else str(inicio)
        no_app = next((c for c in cartas if c['rollMin'] == inicio and c['rollMax'] == fim), None)
        print(f'{faixa:>7} {nome:16} {no_app["card"] if no_app else "— AUSENTE"}')
        if no_app is None:
            falhas.append(f'carta {nome} ({faixa}): o app não tem nenhuma carta nessa faixa')
        elif P.chave(no_app['card']) != P.chave(nome):
            falhas.append(f'a faixa {faixa} é {nome} no livro e {no_app["card"]!r} no app')
    if len(cartas) != len(linhas):
        falhas.append(f'o app tem {len(cartas)} cartas e a p347 imprime {len(linhas)}')
    print(f'\n{len(cartas)}/{len(linhas)} cartas, d% 1-100 tilado')
    return falhas


def imprime_as_notas(no_app: dict):
    """A nota de `unmodeled` de cada artefato, para o OLHO conferir se envelheceu."""
    print('\n══ O QUE NÃO FOI MODELADO ' + '═' * 34)
    for _chave, entrada in sorted(no_app.items(), key=lambda kv: kv[1]['name']):
        nota = entrada.get('unmodeled') or '(nada: o motor aplica o verbete inteiro)'
        print(f'\n  {entrada["name"]} (p{entrada.get("bookPage")})')
        for linha in P.junta([nota]).split('. '):
            print(f'    {linha.strip()}')


if __name__ == '__main__':
    raise SystemExit(min(1, audita()))

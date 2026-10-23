#!/usr/bin/env python3
"""Confere os ENCANTOS do catálogo contra o livro (ALE-411, ALE-439).

São DUAS TABELAS e UM auditor
-----------------------------
A Tabela 8-8 é de armas (p335-336) e a 8-10 é de armaduras e escudos
(p338-339), e as duas têm a mesma FORMA: faixa de `d%` + nome + efeito na
tabela, `Nome. Regra` no verbete, e o catálogo do outro lado. O que muda entre
elas são coordenadas e três constantes.

Por isso elas são PERFIS deste arquivo, e não dois scripts: uma segunda cópia
divergiria na primeira correção de leitura — e correção de leitura é o que este
auditor mais faz. Rode `audit-enchants.py arma` ou `armadura` para uma só.

As duas âncoras, e o que elas conferem uma na outra
---------------------------------------------------
O livro declara os encantos DUAS vezes, e as duas leituras se cobram antes de
qualquer correção no catálogo:

* a **Tabela 8-8** (p336), `d% | Encanto | Efeito`, que é a faixa de rolagem e o
  resumo de uma linha;
* o **verbete** (p335-336), na forma `Nome. Regra`, que é a regra inteira.

Um nome que aparece numa e não na outra é defeito de LEITURA, e ele sai antes de
o catálogo ser mencionado — foi assim que o `audit-origins.py` descobriu que a
página saía fora de ordem.

O denominador é de COBERTURA, e a diferença importa
---------------------------------------------------
Nos auditores da ALE-391 havia duas coisas para comparar: o livro e um catálogo
que já existia. Aqui o catálogo COMEÇA vazio, então "transcrevi os encantos" e
"transcrevi doze dos vinte e oito" têm a mesma cara no terminal. Por isso o
relatório afirma, sempre: quantos a tabela tem, quantos entraram, e quantos
ficaram sem modificador — com o MOTIVO de cada um.

Um encanto com modificador não está pronto por isso
---------------------------------------------------
Quase todo encanto tem duas metades: o Flamejante é "+1d6 de fogo" E "gaste 2 PM
para disparar uma bola de fogo". Medir só "tem modifiers?" daria verde a quem
modelou 30% da regra. A terceira dimensão é a COBERTURA de palavras do verbete
pela descrição do catálogo, que é o instrumento que a ALE-405 teve de inventar
quando os números coincidiram por acaso.
"""
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import t20pdf as P

# O livro impresso está 6 páginas atrás do PDF, e a conta não é repetida aqui de
# propósito: ela é do `engine-go/CLAUDE.md`, que é o dono dela.
OFFSET_DO_PDF = 6

# A ÚLTIMA LINHA de cada tabela vem COLAPSADA num `<line>` só — "91-100 Arma
# específica Veja a Tabela 8-9" —, e é por isso que o padrão aceita cauda.
RE_FAIXA = re.compile(r'^(\d{1,3})(?:-(\d{1,3}))?(?:\s+(.*))?$')
# O verbete começa com o nome em NEGRITO seguido de ponto. O `[A-ZÁÉÍÓÚÂÊÔÃÕÇ]`
# na primeira letra é o que separa "Flamejante. A arma…" de uma frase qualquer
# que termine em nome próprio.
RE_VERBETE = re.compile(r'^([A-ZÁÉÍÓÚÂÊÔÃÕÇ][a-záéíóúâêôãõç]+)\. (.+)$')

FOLGA = 3.0


class Perfil:
    """Uma das duas tabelas de encanto, com onde ela mora na página.

    Tudo aqui foi MEDIDO no PDF, coluna por coluna. O `x` da célula do dado é
    uma FAIXA e não um ponto porque a coluna é alinhada à DIREITA: na 8-8,
    "01-05" começa em 59,1 e "46" em 65,1.
    """

    def __init__(self, *, chave, tabela, pagina_da_tabela, paginas_dos_verbetes,
                 x_do_dado, x_do_nome, x_do_efeito, colunas_da_prosa,
                 regiao_da_tabela, nao_e_encanto, contam_dobrado, fim_dos_verbetes,
                 categoria):
        self.chave = chave
        self.tabela = tabela
        self.pagina_da_tabela = pagina_da_tabela
        self.paginas_dos_verbetes = paginas_dos_verbetes
        self.x_do_dado = x_do_dado
        self.x_do_nome = x_do_nome
        self.x_do_efeito = x_do_efeito
        self.colunas_da_prosa = colunas_da_prosa
        self.regiao_da_tabela = regiao_da_tabela
        self.nao_e_encanto = nao_e_encanto
        self.contam_dobrado = contam_dobrado
        self.fim_dos_verbetes = fim_dos_verbetes
        self.categoria = categoria


PERFIS = {
    # ARMAS: a Tabela 8-8 mora na p336 e os verbetes atravessam as duas páginas.
    #
    # A tabela é lida SEM o `sem_lixo`, que é a armadilha desta página: o padrão
    # de número de página (`^\d{1,3}$`) casa com as faixas de UM valor, e a
    # limpeza comeu a Energética (46) e a Lancinante (64). Saíram 26 linhas com
    # os nomes certos, as faixas certas e dois efeitos colados no vizinho de
    # cima — o que denunciou foi a soma das faixas não fechar em 100.
    #
    # E na p336 o X NÃO BASTA: a prosa da coluna esquerda começa em 51 e a
    # célula de d% cai entre 57 e 68 — as duas se sobrepõem. O que as separa é a
    # ALTURA: a tabela acaba em y=527 (a nota do asterisco) e a prosa de baixo
    # começa em y=596. Recortar só por x perdia três verbetes, entre eles o da
    # Magnífica, e o relatório saía com 25 de 28 sem dizer que tinha perdido.
    'arma': Perfil(
        chave='arma', tabela='8-8',
        pagina_da_tabela=342, paginas_dos_verbetes=(341, 342),
        x_do_dado=(56.0, 67.0), x_do_nome=90.8, x_do_efeito=154.3,
        # Cada par é (corpo, primeira linha indentada), e o degrau é de 17 pontos.
        colunas_da_prosa={341: (71.0, 88.0, 309.0, 326.0), 342: (51.0, 68.0, 289.0, 306.0)},
        regiao_da_tabela={342: (200.0, 540.0)},
        # A Tabela 8-8 tem 29 linhas de dado e 28 encantos: "91-100 Arma
        # específica" manda rolar em OUTRA tabela, e o livro diz que ela apaga
        # os encantos rolados. Declarada para o denominador não subir para 29 em
        # silêncio nem descer para 28 sem motivo.
        nao_e_encanto='Arma específica',
        # O ASTERISCO: "*Conta como dois encantos."
        contam_dobrado={'Energética', 'Lancinante', 'Magnífica'},
        # ONDE OS VERBETES ACABAM. O último (Venenosa) não tem sucessor que o
        # pare, e sem esta linha ele engolia a seção seguinte inteira — "Armas
        # Específicas" e o "Arco do Poder" viravam parte da regra dele, inflando
        # a cobertura com palavras de outro assunto.
        fim_dos_verbetes='Armas Específicas',
        categoria='weapon-enchant',
    ),
    # ARMADURAS E ESCUDOS: a Tabela 8-10 mora na p339, na coluna da direita, e
    # os verbetes começam na p338 (coluna direita, depois do cabeçalho da seção)
    # e terminam na p339 com o Zeloso — que é a última coisa da página, e por
    # isso não há sentinela de fim aqui.
    #
    # O x da prosa (309/326) e o da célula de dado (317) convivem sem se
    # confundir: oito pontos de distância contra uma folga de três. É por isso
    # que este perfil não precisa da região por altura que o de arma precisa.
    'armadura': Perfil(
        chave='armadura', tabela='8-10',
        pagina_da_tabela=345, paginas_dos_verbetes=(344, 345),
        x_do_dado=(313.0, 319.0), x_do_nome=348.7, x_do_efeito=412.2,
        colunas_da_prosa={344: (289.1, 306.1), 345: (70.9, 87.9, 309.0, 326.0)},
        regiao_da_tabela={},
        nao_e_encanto='Item específico',
        contam_dobrado={'Guardião'},
        fim_dos_verbetes=None,
        categoria='armor-enchant',
    ),
}

def linha_da_tabela(perfil: Perfil):
    """As linhas da tabela do perfil: (inicio, fim, nome, efeito).

    A linha se monta pelo Y — as três células são blocos DIFERENTES, e o que
    diz que "01-05", "Ameaçadora" e "Duplica margem de ameaça" são a mesma linha
    é a altura das três ser igual. Parear por índice erra na primeira célula que
    quebra em duas linhas, e a da Caçadora quebra.
    """
    celulas = P.linhas_com_coordenada(perfil.pagina_da_tabela)
    dados = []
    for x, y, texto in celulas:
        if not perfil.x_do_dado[0] <= x <= perfil.x_do_dado[1]:
            continue
        casa = RE_FAIXA.match(texto)
        if casa:
            dados.append((y, casa.groups()))
    nomes = [(y, t) for x, y, t in celulas if abs(x - perfil.x_do_nome) < FOLGA]
    efeitos = sorted((y, t) for x, y, t in celulas if abs(x - perfil.x_do_efeito) < FOLGA)

    for y, (inicio, fim, cauda) in sorted(dados):
        # A linha colapsada traz nome e efeito na própria cauda, e os dois se
        # separam pela tabela ser a última: "Arma específica Veja a Tabela 8-9".
        if cauda:
            nome, _, efeito = cauda.partition(' Veja ')
            yield int(inicio), int(fim or inicio), nome.strip(), ('Veja ' + efeito).strip()
            continue
        nome = next((t for yn, t in nomes if abs(yn - y) < P.MESMA_LINHA), None)
        if nome is None:
            continue
        # O efeito vai da altura desta linha até a da próxima: a célula que
        # quebra em duas (a Caçadora) tem a continuação SEM d% ao lado.
        proxima = next((yd for yd, _ in sorted(dados) if yd > y + P.MESMA_LINHA), 1e9)
        corpo = [t for ye, t in efeitos if y - P.MESMA_LINHA < ye < proxima - P.MESMA_LINHA]
        nome, efeito = nome_e_efeito(nome, P.junta(corpo))
        yield int(inicio), int(fim or inicio), nome, efeito


# A CHAMADA DE RODAPÉ vem de dois jeitos na mesma tabela, e os dois são o mesmo
# "1" ou "2": célula PRÓPRIA ao lado do nome (o Animado e o Guardião da 8-10) ou
# GRUDADA nele ("Esmagador 1"). O asterisco da 8-8 é a terceira grafia.
#
# Nenhum encanto do livro termina em dígito, então tirar o sufixo é seguro — e
# se um dia terminar, as duas âncoras discordam e o auditor para antes de
# encostar no catálogo.
RE_CHAMADA = re.compile(r'[\s*]*\d?$')


def nome_e_efeito(celula: str, efeito: str):
    """Separa o nome do efeito quando a página os entrega na MESMA célula.

    Duas linhas da Tabela 8-10 vêm assim — "Fantasmagórico Lança Manto de
    Sombras" —, e nelas não existe célula de efeito na altura da linha. O corte
    é na primeira palavra porque todo encanto das duas tabelas tem nome de uma
    palavra só; se um dia não tiver, o nome cortado não vai casar com o verbete
    e o auditor para na comparação das âncoras, que é onde ele tem de parar.
    """
    nome = RE_CHAMADA.sub('', celula).strip()
    if efeito or ' ' not in nome:
        return nome, efeito
    primeira, _, resto = nome.partition(' ')
    return primeira, resto


def faixas_que_nao_fecham(linhas) -> str:
    """A SOMA antes das parcelas: as faixas de d% têm de ladrilhar 1..100.

    É o denominador de graça de toda decomposição, e é o único controle que
    pega uma linha SUMIDA: 26 linhas com nomes certos e efeitos plausíveis se
    parecem com 29 no terminal, e a única coisa que não fecha é o dado.
    """
    esperado = 1
    for inicio, fim, nome, _ in linhas:
        if inicio != esperado:
            return (f'a faixa de {nome} começa em {inicio} e a anterior terminou '
                    f'em {esperado - 1}: falta linha entre as duas')
        esperado = fim + 1
    if esperado != 101:
        return f'as faixas terminam em {esperado - 1} e o d% vai até 100'
    return ''


def verbetes(perfil: Perfil):
    """Os verbetes `Nome. Regra` das páginas, com a regra inteira.

    Um verbete termina onde o seguinte começa. A prosa dos encantos divide a
    página com a tabela, então a leitura tem de ser por COLUNA — é o que o
    `P.leitura` entrega.
    """
    achados: dict[str, list[str]] = {}
    pagina_de: dict[str, int] = {}
    atual = None
    for pagina in perfil.paginas_dos_verbetes:
        prosa = perfil.colunas_da_prosa[pagina]
        limite = perfil.regiao_da_tabela.get(pagina)
        for x, y, texto in P.leitura(pagina):
            if not any(abs(x - coluna) < FOLGA for coluna in prosa):
                continue
            if limite and x < limite[0] and y < limite[1]:
                continue
            if perfil.fim_dos_verbetes and texto.startswith(perfil.fim_dos_verbetes):
                atual = None
                continue
            casa = RE_VERBETE.match(texto)
            if casa and casa.group(1) not in achados:
                atual = casa.group(1)
                achados[atual] = [casa.group(2)]
                pagina_de[atual] = pagina - OFFSET_DO_PDF
            elif atual is not None:
                achados[atual].append(texto)
    return {nome: (P.junta(corpo), pagina_de[nome]) for nome, corpo in achados.items()}


# O PISO DA COBERTURA. A descrição do catálogo é TRANSCRITA do verbete, então
# ela nasce em 100% — e é por isso que o piso é alto. O que este número prende
# não é a transcrição de hoje: é o dia em que alguém "melhorar" a frase e ela
# passar a dizer outra coisa, que foi o defeito da ALE-408 em dois arquivos de
# Poder Concedido.
PISO_DA_COBERTURA = 0.95

CATALOGO = P.RAIZ / 'engine-go/domain/catalog/data/items.json'


def do_catalogo(categoria: str) -> dict:
    itens = json.loads(CATALOGO.read_text())
    return {i['name']: i for i in itens if i.get('category') == categoria}


def audita(perfil: Perfil) -> int:
    """Audita UMA tabela e devolve quantas falhas achou."""
    print(f'\n══ TABELA {perfil.tabela} ({perfil.chave}) ' + '═' * 40)
    tabela = list(linha_da_tabela(perfil))
    falhas = []

    # ── ÂNCORA 1 contra ÂNCORA 2, antes de o catálogo ser mencionado ──────
    buraco = faixas_que_nao_fecham(tabela)
    if buraco:
        print(f'A TABELA {perfil.tabela} NÃO FECHA: {buraco}')
        print('Nada do que vem abaixo vale — o auditor leu a página errado.')
        return 1

    da_tabela = {nome: (inicio, fim, efeito)
                 for inicio, fim, nome, efeito in tabela if nome != perfil.nao_e_encanto}
    dos_verbetes = verbetes(perfil)

    print(f'AS DUAS ÂNCORAS  tabela: {len(tabela)} linhas (soma fecha em 100), '
          f'{len(da_tabela)} encantos + "{perfil.nao_e_encanto}"')
    print(f'                 verbetes: {len(dos_verbetes)}')
    so_na_tabela = sorted(set(da_tabela) - set(dos_verbetes))
    so_no_verbete = sorted(set(dos_verbetes) - set(da_tabela))
    if so_na_tabela or so_no_verbete:
        print(f'  AS DUAS DISCORDAM — só na tabela: {so_na_tabela}; '
              f'só no verbete: {so_no_verbete}')
        print('  É defeito de LEITURA, e ele vem antes de qualquer correção de catálogo.')
        return 1
    print(f'                 as duas dizem os MESMOS {len(da_tabela)} nomes')

    # ── O CATÁLOGO ────────────────────────────────────────────────────────
    catalogo = do_catalogo(perfil.categoria)
    print(f'\nCATÁLOGO         {len(catalogo)} entradas em `{perfil.categoria}`')

    com_modificador, sem_modificador, cobertura_minima = 0, [], (2.0, '')
    em_parte = []
    faltando = []
    for nome, (inicio, fim, _efeito) in sorted(da_tabela.items()):
        entrada = catalogo.get(nome)
        if entrada is None:
            faltando.append(nome)
            falhas.append(f'{nome}: está na Tabela {perfil.tabela} e não está no catálogo')
            continue
        regra, pagina = dos_verbetes[nome]

        quanto, juntas, total = P.cobertura(regra, entrada.get('description', ''))
        cobertura_minima = min(cobertura_minima, (quanto, nome))
        if quanto < PISO_DA_COBERTURA:
            falhas.append(
                f'{nome}: a descrição do catálogo repete {juntas} das {total} palavras '
                f'de conteúdo do verbete ({quanto:.0%}) — abaixo do piso de '
                f'{PISO_DA_COBERTURA:.0%}. Ela deixou de dizer a regra do livro.')

        if (entrada.get('rollMin'), entrada.get('rollMax')) != (inicio, fim):
            falhas.append(
                f'{nome}: o catálogo diz d% {entrada.get("rollMin")}-{entrada.get("rollMax")} '
                f'e a Tabela {perfil.tabela} diz {inicio}-{fim}')
        esperado = 2 if nome in perfil.contam_dobrado else 1
        if entrada.get('countsAs') != esperado:
            falhas.append(
                f'{nome}: o catálogo conta como {entrada.get("countsAs")} e o livro diz '
                f'{esperado} (o asterisco da tabela é "conta como dois encantos")')
        if entrada.get('bookPage') != pagina:
            falhas.append(f'{nome}: o catálogo cita a p{entrada.get("bookPage")} e o '
                          f'verbete está na p{pagina}')

        # O DENOMINADOR do que o motor aplica, em TRÊS baldes e não dois.
        #
        # Aqui ter modificador e ter motivo eram mutuamente exclusivos, pela
        # premissa de que um encanto cabe inteiro no motor ou não cabe nada
        # dele. Ela é falsa em dez dos vinte e oito — a Flamejante soma o +1d6
        # de fogo E tem um gesto de 6d6 que o motor não resolve —, e o efeito
        # da exclusividade era SILÊNCIO: escrever a nota honesta reprovava,
        # então a metade que não entrou não era escrita (ALE-459).
        #
        # Quem aplica EM PARTE diz as duas coisas, e é o `modeled` que impede a
        # nota de virar desculpa: ele nomeia a metade que entrou.
        tem_modelo = bool(entrada.get('modifiers'))
        motivo = entrada.get('unmodeled', '')
        modelado = entrada.get('modeled') or []
        if tem_modelo and motivo:
            em_parte.append((nome, modelado, motivo))
        elif tem_modelo:
            com_modificador += 1
        else:
            sem_modificador.append((nome, motivo))
        if not tem_modelo and not motivo:
            falhas.append(f'{nome}: não tem modificador NEM diz por quê')
        if tem_modelo and motivo and not modelado:
            falhas.append(
                f'{nome}: aplica alguma coisa, diz o que NÃO entrou ({motivo!r}) e não '
                f'diz o que ENTROU — sem o `modeled` a nota parece falar do encanto inteiro')

    print(f'  no catálogo:     {len(da_tabela) - len(faltando)} de {len(da_tabela)}')
    if faltando:
        print(f'  FALTAM {len(faltando)}: {", ".join(faltando)}')
    print(f'  INTEIROS:        {com_modificador}')
    print(f'  EM PARTE:        {len(em_parte)}, e cada um diz as duas metades:')
    for nome, modelado, motivo in em_parte:
        print(f'    {nome:16s} entrou: {"; ".join(modelado)}')
        print(f'    {"":16s}   ficou: {motivo}')
    print(f'  SEM modificador: {len(sem_modificador)}, e cada um diz por quê:')
    for nome, motivo in sem_modificador:
        print(f'    {nome:16s} {motivo}')
    if cobertura_minima[1]:
        print(f'\n  cobertura mínima: {cobertura_minima[0]:.0%} ({cobertura_minima[1]})')

    print(f'\nFALHAS: {len(falhas)}   (medidos: {len(da_tabela)} encantos, '
          f'{len(tabela)} linhas de tabela, {len(dos_verbetes)} verbetes)')
    for f in falhas:
        print(f'  {f}')
    return len(falhas)


def main() -> int:
    escolhidos = sys.argv[1:] or list(PERFIS)
    desconhecidos = [c for c in escolhidos if c not in PERFIS]
    if desconhecidos:
        print(f'não conheço {desconhecidos}; os perfis são {list(PERFIS)}')
        return 1
    return 1 if sum(audita(PERFIS[c]) for c in escolhidos) else 0


if __name__ == '__main__':
    sys.exit(main())

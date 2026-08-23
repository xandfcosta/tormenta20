export type {
  ChaseEventRow,
  ClassTrainedExpertises,
  DevotoTerms,
  DungeonDesign,
  DungeonIdea,
  DungeonSizeRow,
  GmTables,
  RewardCastigoRow,
  RollRangeRow,
  RuinaRow,
} from './rules-tables'
import type {
  ClassTrainedExpertises,
  DevotoTerms,
  DungeonDesign,
  GmTables,
} from './rules-tables'
/**
 * HTTP client for the Go backend. Framework-agnostic on purpose — this file
 * carries no Solid import and ports ~1:1 from the React app. It grows one
 * domain at a time as scenes are ported (ALE-63).
 *
 * Only the READ endpoints the query layer needs are here; each scene brings its
 * own mutations when it lands, so the client never gets ahead of its consumers.
 */
import type { ActivationSpec, CatalogSpell, ClassPower, Condition, ConditionId, Deus, GeneralPower, GrantedPower, GrantedPowerOption, Monster, Origem, OriginDefinition, Raca, TormentaPower, TormentaPowerId } from '@/shared/api/catalog-types'
import type { CampaignCreature, CreatureInput } from '@/shared/api/creature-types'
import type { CatalogItem } from '@/shared/api/item-types'
import type {
  AddMemberInput,
  ApplyDamageResult,
  ApplyEffectInput,
  ApplyEffectResult,
  CastSpellResult,
  AccountInvite,
  AdminUser,
  AuthUser,
  Backup,
  Campaign,
  CampaignInvitePreview,
  CampaignInviteToken,
  CampaignMember,
  Character,
  CharacterExpertise,
  PasswordResetTarget,
  ServerStatus,
  CharacterItem,
  CharacterOptions,
  CharacterSpell,
  ClassLevelResult,
  AbilityChoicesResult,
  ConditionsResult,
  PlayState,
  PowerUseScope,
  ConsumeItemInput,
  ConsumeItemResult,
  ComputedSheetV2,
  EffectsClearedResult,
  CreateCampaignInput,
  CreateExpertiseInput,
  CreateCharacterInput,
  CreateItemInput,
  CreateSessionInput,
  ProficienciesResult,
  Credentials,
  RaceDefinition,
  Session,
  UpdateCampaignInput,
  UpdateExpertiseInput,
  UpdateAbilityChoicesInput,
  UpdateClassLevelInput,
  UpdateItemInput,
  UpdateProficienciesInput,
  UpdateSessionInput,
  UpdateTibarInput,
  TibarResult,
  SpellAugmentPick,
  UnlearnSpellResult,
  UpdateVitalsInput,
  VitalsResult,
} from './types'

export * from './types'

export type FieldErrorMap = Record<string, string[]>

export class ApiError extends Error {
  status: number
  fieldErrors: FieldErrorMap

  constructor(status: number, message: string, fieldErrors: FieldErrorMap = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.fieldErrors = fieldErrors
  }
}

// Vite proxies /api to the Go server (:3001), stripping the prefix.
const API_BASE = '/api'

type ErrorBody = {
  message?: string | string[]
  fieldErrors?: FieldErrorMap
}

/** Reads the backend's error shape, falling back to the bare HTTP status. */
async function toApiError(res: Response): Promise<ApiError> {
  const body = (await res.json().catch(() => null)) as ErrorBody | null
  const raw = body?.message
  const message = Array.isArray(raw) ? raw.join('; ') : (raw ?? `${res.status} ${res.statusText}`)
  return new ApiError(res.status, message, body?.fieldErrors ?? {})
}

function createRequest(fetchImpl: typeof globalThis.fetch) {
  return async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetchImpl(`${API_BASE}${path}`, {
      ...init,
      // The session is an httpOnly cookie — without this every call is a 401.
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
    })
    if (!res.ok) throw await toApiError(res)
    if (res.status === 204) return undefined as T
    return res.json() as Promise<T>
  }
}

/**
 * Builds a client over an injected `fetch`, so tests drive it with a FakeFetch
 * instead of patching a global.
 *
 * @example const api = createApiClient(new FakeFetch([...]).fetch)
 */
export function createApiClient(fetchImpl: typeof globalThis.fetch = globalThis.fetch) {
  const request = createRequest(fetchImpl)
  const json = (body: unknown): RequestInit => ({
    method: 'POST',
    body: JSON.stringify(body),
  })
  const patch = (body: unknown): RequestInit => ({
    method: 'PATCH',
    body: JSON.stringify(body),
  })
  const put = (body: unknown): RequestInit => ({
    method: 'PUT',
    body: JSON.stringify(body),
  })
  const del: RequestInit = { method: 'DELETE' }

  return {
    auth: {
      /**
       * `inviteToken` is required for everyone but the ADMIN_EMAILS addresses —
       * the server answers 403 without a usable one (ALE-120).
       */
      register: (input: Credentials & { name?: string; inviteToken?: string }) =>
        request<AuthUser>('/auth/register', json(input)),
      login: (input: Credentials) => request<AuthUser>('/auth/login', json(input)),
      logout: () => request<void>('/auth/logout', { method: 'POST' }),
      /** Anônimo: quem esqueceu a senha não consegue autenticar para trocá-la. */
      resetPassword: (input: { token: string; password: string }) =>
        request<void>('/auth/reset-password', json(input)),
      me: () => request<AuthUser>('/auth/me'),
    },
    characters: {
      list: () => request<Character[]>('/characters'),
      get: (id: number) => request<Character>(`/characters/${id}`),
      /** The creation wizard's pick lists (races, classes, origins…). Static. */
      options: () => request<CharacterOptions>('/characters/options'),
      /** Forges the character — the Forja's one write. */
      create: (input: CreateCharacterInput) =>
        request<Character>('/characters', json(input)),
      /** The server-computed sheet (defense, attribute totals, perícias…). */
      getSheet: (id: number) => request<ComputedSheetV2>(`/characters/${id}/sheet`),
      /** Trains/untrains a perícia or rekeys it to another attribute. */
      updateExpertise: (id: number, input: UpdateExpertiseInput) =>
        request<CharacterExpertise>(`/characters/${id}/expertises`, patch(input)),
      /** A custom "ofício" the player invented. */
      addExpertise: (id: number, input: CreateExpertiseInput) =>
        request<CharacterExpertise>(`/characters/${id}/expertises`, json(input)),
      deleteExpertise: (id: number, name: string) =>
        request<{ name: string }>(
          `/characters/${id}/expertises/${encodeURIComponent(name)}`,
          del,
        ),
      addItem: (id: number, input: CreateItemInput) =>
        request<CharacterItem>(`/characters/${id}/items`, json(input)),
      updateItem: (id: number, itemId: number, input: UpdateItemInput) =>
        request<CharacterItem>(`/characters/${id}/items/${itemId}`, patch(input)),
      deleteItem: (id: number, itemId: number) =>
        request<{ id: number }>(`/characters/${id}/items/${itemId}`, del),
      /** Spends one unit; the server answers a DELTA, not the whole character. */
      consumeItem: (id: number, itemId: number, input?: ConsumeItemInput) =>
        request<ConsumeItemResult>(`/characters/${id}/items/${itemId}/consume`, json(input ?? {})),
      /** PV/PM edit. Answers the CLAMPED pair, so the client takes the server's word. */
      updateVitals: (id: number, input: UpdateVitalsInput) =>
        request<VitalsResult>(`/characters/${id}/vitals`, patch(input)),
      /** O dinheiro do personagem (T$). Saldo inteiro, não delta. */
      updateTibar: (id: number, input: UpdateTibarInput) =>
        request<TibarResult>(`/characters/${id}/tibar`, patch(input)),
      /**
       * Damage in ONE request: the server routes it temp-PV-first and answers
       * what it drained. Two requests (drain, then subtract) could interleave
       * with another client's hit and lose a pool.
       */
      applyDamage: (id: number, amount: number) =>
        request<ApplyDamageResult>(`/characters/${id}/damage`, json({ amount })),
      /** Replaces the proficiency set (weapons, armor, shields). */
      updateProficiencies: (id: number, input: UpdateProficienciesInput) =>
        request<ProficienciesResult>(`/characters/${id}/proficiencies`, patch(input)),
      /** Level one class up or down; answers the new totals AND the resynced
       *  PV/PM pools, which are derived from class levels. */
      updateClassLevel: (id: number, input: UpdateClassLevelInput) =>
        request<ClassLevelResult>(`/characters/${id}/classes/level`, patch(input)),
      /** Spell buff, power grant, or the GM's manual temp-PV pool. */
      applyEffect: (id: number, input: ApplyEffectInput) =>
        request<ApplyEffectResult>(`/characters/${id}/active-effects`, json(input)),
      removeActiveEffect: (id: number, effectId: number) =>
        request<{ id: number }>(`/characters/${id}/active-effects/${effectId}`, del),
      /** Expires the scene-scoped effects. The player's own "Encerrar cena" —
       *  the GM's session-wide rest reaches the same rule over the WS. */
      endScene: (id: number) =>
        request<EffectsClearedResult>(`/characters/${id}/end-scene`, { method: 'POST' }),
      endDay: (id: number) =>
        request<EffectsClearedResult>(`/characters/${id}/end-day`, { method: 'POST' }),
      /** Patches ANY SUBSET of the five ability-choice blobs (poderes de
       *  classe, escolhas de origem/raça, caminhos). Sending none is a 400. */
      updateAbilityChoices: (id: number, input: UpdateAbilityChoicesInput) =>
        request<AbilityChoicesResult>(`/characters/${id}/abilities`, patch(input)),
      /** Adds a spell to the grimoire, unprepared. 409 if already known. */
      learnSpell: (id: number, catalogSpellId: string) =>
        request<CharacterSpell>(`/characters/${id}/spells`, json({ catalogSpellId })),
      unlearnSpell: (id: number, catalogSpellId: string) =>
        request<UnlearnSpellResult>(
          `/characters/${id}/spells/${encodeURIComponent(catalogSpellId)}`,
          del,
        ),
      /** 404 (not 400) when the spell isn't learned, so the UI can say
       *  "aprenda primeiro" instead of a generic failure. */
      setSpellPrepared: (id: number, catalogSpellId: string, prepared: boolean) =>
        request<CharacterSpell>(
          `/characters/${id}/spells/${encodeURIComponent(catalogSpellId)}/prepared`,
          patch({ prepared }),
        ),
      /** Answers a DELTA (new PM), not the character. The server re-validates
       *  learned/prepared/augments and the per-spell PM limit. */
      castSpell: (id: number, catalogSpellId: string, augments: SpellAugmentPick[] = []) =>
        request<CastSpellResult>(
          `/characters/${id}/spells/${encodeURIComponent(catalogSpellId)}/cast`,
          json({ augments }),
        ),
      /** Replaces the active book conditions (caído, atordoado…, p394-395). */
      updateConditions: (id: number, activeConditions: ConditionId[]) =>
        request<ConditionsResult>(
          `/characters/${id}/conditions`,
          patch({ activeConditions }),
        ),

      /**
       * O estado de JOGO da ficha (ALE-222). Os quatro devolvem o estado
       * INTEIRO, para a tela conferir o próprio otimismo contra o servidor.
       *
       * CUIDADO com o vizinho de cima: `conditionals` é o opt-in do JOGADOR;
       * `conditions` são as do LIVRO. Ver C6 no GLOSSARIO.md.
       */
      updateConditionals: (id: number, conditionals: string[]) =>
        request<PlayState>(`/characters/${id}/conditionals`, patch({ conditionals })),

      /**
       * Gasta MAIS UM uso. O corpo não carrega o total de propósito: dois
       * cliques rápidos mandando "agora são 3" gravariam 3 duas vezes e
       * perderiam um uso — o servidor soma no `ON CONFLICT`.
       */
      bumpPowerUse: (id: number, powerId: string, scope: PowerUseScope) =>
        request<PlayState>(`/characters/${id}/power-uses`, json({ powerId, scope })),

      /** Registra o que foi pago para entrar na postura. */
      setStance: (id: number, flag: string, payment: { steps: number; pmPaid: number }) =>
        request<PlayState>(
          `/characters/${id}/stances/${encodeURIComponent(flag)}`,
          put(payment),
        ),

      /** Esquece o pagamento — sair da postura. */
      clearStance: (id: number, flag: string) =>
        request<PlayState>(`/characters/${id}/stances/${encodeURIComponent(flag)}`, del),
    },
    catalog: {
      // Static rulebook reference; cached hard (staleTime ∞) and fetched instead
      // of bundled — the front ships no catalog DATA
      // (project_front_decouple_catalog).
      spells: () => request<Record<string, CatalogSpell>>('/catalog/spells'),
      items: () => request<CatalogItem[]>('/catalog/items'),
      /** The bestiary the Mesa do Mestre and the in-session monster add draw
       *  from — 80 creatures, served rather than bundled like every catalog. */
      bestiary: () => request<Monster[]>('/catalog/bestiary'),
      /** RaceDefinition catalog (innate abilities) — DISTINCT from `races`. */
      raceDefs: () => request<RaceDefinition[]>('/catalog/race-defs'),
      origins: () => request<OriginDefinition[]>('/catalog/origins'),
      classPowers: () => request<ClassPower[]>('/catalog/class-powers'),
      generalPowers: () => request<GeneralPower[]>('/catalog/general-powers'),
      deuses: () => request<Deus[]>('/catalog/deuses'),
      grantedPowers: () => request<GrantedPower[]>('/catalog/granted-powers'),
      /** racas.ts RACAS (movement/size/attr-mod) — DISTINCT from `raceDefs`. */
      races: () => request<Record<string, Raca>>('/catalog/races'),
      origens: () => request<Record<string, Origem>>('/catalog/origens'),
      conditions: () => request<Record<ConditionId, Condition>>('/catalog/conditions'),
      tormentaPowers: () =>
        request<Record<TormentaPowerId, TormentaPower>>('/catalog/tormenta-powers'),
      divinePowers: () => request<GrantedPowerOption[]>('/catalog/divine-powers'),
      activations: () => request<ActivationSpec[]>('/catalog/activations'),
      // Tabelas de livro autoradas no servidor (ALE-102): eram as últimas que o
      // front importava em tempo de build.
      classExpertises: () =>
        request<Record<string, ClassTrainedExpertises>>('/catalog/class-expertises'),
      devotoTerms: () => request<DevotoTerms>('/catalog/devoto-terms'),
      gmTables: () => request<GmTables>('/catalog/gm-tables'),
      dungeonDesign: () => request<DungeonDesign>('/catalog/dungeon-design'),
    },
    campaigns: {
      list: () => request<Campaign[]>('/campaigns'),
      get: (id: number) => request<Campaign>(`/campaigns/${id}`),
      create: (input: CreateCampaignInput) => request<Campaign>('/campaigns', json(input)),
      update: (id: number, input: UpdateCampaignInput) =>
        request<Campaign>(`/campaigns/${id}`, patch(input)),
      delete: (id: number) => request<{ id: number }>(`/campaigns/${id}`, del),
      /** Mints a fresh invite token — any link shared before this 404s. */
      rotateInvite: (id: number) =>
        request<CampaignInviteToken>(`/campaigns/${id}/invite`, { method: 'POST' }),
    },
    /**
     * Blocos de criatura que o MESTRE escreveu para a campanha (ALE-137). O
     * servidor responde 403 ao jogador até no GET: o bloco é informação do
     * mestre, e o jogador vê a criatura pela iniciativa — nome e barra de PV,
     * com a regra de PV oculto.
     */
    creatures: {
      list: (campaignId: number) =>
        request<CampaignCreature[]>(`/campaigns/${campaignId}/creatures`),
      create: (campaignId: number, input: CreatureInput) =>
        request<CampaignCreature>(`/campaigns/${campaignId}/creatures`, json(input)),
      update: (campaignId: number, id: number, input: CreatureInput) =>
        request<CampaignCreature>(`/campaigns/${campaignId}/creatures/${id}`, patch(input)),
      delete: (campaignId: number, id: number) =>
        request<void>(`/campaigns/${campaignId}/creatures/${id}`, del),
    },
    invites: {
      /** Public: resolves a shared token to the campaign it invites into. */
      resolve: (token: string) =>
        request<CampaignInvitePreview>(`/invites/${encodeURIComponent(token)}`),
    },
    admin: {
      users: () => request<AdminUser[]>('/admin/users'),
      /** Apaga a conta; as mesas dela passam para quem apagou. */
      deleteUser: (id: number) =>
        request<{ id: number; transferredCampaigns: number }>(`/admin/users/${id}`, del),
      /** Gera o link de uso único; o jogador escolhe a própria senha. */
      passwordReset: (id: number) =>
        request<AccountInvite>(`/admin/users/${id}/password-reset`, { method: 'POST' }),
      invites: () => request<AccountInvite[]>('/admin/invites'),
      status: () => request<ServerStatus>('/admin/status'),
      backups: () => request<Backup[]>('/admin/backups'),
      createBackup: () => request<Backup>('/admin/backups', { method: 'POST' }),
    },
    passwordResets: {
      /** Público: diz de quem é o link, ou 404 se ele não serve mais. */
      resolve: (token: string) =>
        request<PasswordResetTarget>(`/password-resets/${encodeURIComponent(token)}`),
    },
    accountInvites: {
      /** Admin only: mints the link that lets ONE person create an account. */
      create: () => request<AccountInvite>('/admin/invites', { method: 'POST' }),
      /** Public: says whether a link is still good, before any account exists. */
      resolve: (token: string) =>
        request<AccountInvite>(`/account-invites/${encodeURIComponent(token)}`),
    },
    members: {
      list: (campaignId: number) => request<CampaignMember[]>(`/campaigns/${campaignId}/members`),
      /** Self-join: the caller must own `characterId`. */
      add: (campaignId: number, input: AddMemberInput) =>
        request<CampaignMember>(`/campaigns/${campaignId}/members`, json(input)),
      /** `id` is the MEMBER's id, not the character's. */
      remove: (campaignId: number, id: number) =>
        request<{ id: number }>(`/campaigns/${campaignId}/members/${id}`, del),
    },
    sessions: {
      list: (campaignId: number) => request<Session[]>(`/campaigns/${campaignId}/sessions`),
      get: (campaignId: number, id: number) =>
        request<Session>(`/campaigns/${campaignId}/sessions/${id}`),
      create: (campaignId: number, input: CreateSessionInput) =>
        request<Session>(`/campaigns/${campaignId}/sessions`, json(input)),
      /** Patches any subset (número, título, notas); answers the whole row. */
      update: (campaignId: number, id: number, input: UpdateSessionInput) =>
        request<Session>(`/campaigns/${campaignId}/sessions/${id}`, patch(input)),
      remove: (campaignId: number, id: number) =>
        request<{ id: number }>(`/campaigns/${campaignId}/sessions/${id}`, del),
      /** planned → active. The live session is the one the Hub resumes. */
      start: (campaignId: number, id: number) =>
        request<Session>(`/campaigns/${campaignId}/sessions/${id}/start`, {
          method: 'POST',
        }),
      end: (campaignId: number, id: number) =>
        request<Session>(`/campaigns/${campaignId}/sessions/${id}/end`, { method: 'POST' }),
    },
  }
}

export type ApiClient = ReturnType<typeof createApiClient>

/** The app-wide client. Tests build their own with `createApiClient`. */
export const api: ApiClient = createApiClient()

import { queryOptions } from '@tanstack/solid-query'
import { type ApiClient, ApiError, type CampaignInvitePreview, api } from '@/shared/api/api'

export const campaignsQueryOptions = queryOptions({
  queryKey: ['campaigns'] as const,
  queryFn: api.campaigns.list,
})

export const campaignQueryOptions = (id: number) =>
  queryOptions({
    queryKey: ['campaigns', id] as const,
    queryFn: () => api.campaigns.get(id),
  })

/**
 * Resolves an invite token, turning "no such invite" into a real failure.
 *
 * The Go backend answers an unknown token with **200 and a `null` body**, not a
 * 404 — so without this an expired link is indistinguishable from one still
 * loading, and the player faces a dead button with no explanation. Verified
 * against the running backend (ALE-80). Takes the client as a parameter so the
 * rule is testable without patching a global.
 *
 * @example await fetchInvitePreview('abc', createApiClient(fake.fetch))
 */
export async function fetchInvitePreview(
  token: string,
  client: ApiClient = api,
): Promise<CampaignInvitePreview> {
  const preview = await client.invites.resolve(token)
  if (!preview?.campaignId) throw new ApiError(404, 'Convite inválido ou expirado')
  return preview
}

/**
 * The public preview behind an invite token. Disabled without a token, and
 * never retried: a dead token is an ANSWER ("peça um link novo"), not a
 * hiccup worth three round-trips.
 */
export const inviteQueryOptions = (token: string | undefined) =>
  queryOptions({
    queryKey: ['invites', token] as const,
    queryFn: () => fetchInvitePreview(token as string),
    enabled: !!token,
    retry: false,
  })

export const campaignMembersQueryOptions = (campaignId: number) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'members'] as const,
    queryFn: () => api.members.list(campaignId),
  })

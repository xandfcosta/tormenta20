export type JoinSource = {
  /** Present when the player arrived through an invite link. */
  token: string | undefined
  /** The campaign the token resolved to — undefined while it's still resolving. */
  invitedCampaignId: number | undefined
  /** What the player typed, when there is no token. */
  typedId: string
}

/**
 * Which campaign the join form is aimed at. The invite wins when there is one;
 * otherwise the number the GM read out loud, and only if it's a real id.
 *
 * This is DERIVED on purpose. The React version mirrored the resolved id into
 * form state through an effect guarded by a `tokenApplied` flag, because
 * writing it during render warned "Cannot update a component while rendering a
 * different component" (ALE-20) — and the mirror could then disagree with the
 * invite. A function of its inputs can't.
 *
 * @example joinTargetId({ token: 'abc', invitedCampaignId: 7, typedId: '3' }) // 7
 */
export function joinTargetId(source: JoinSource): number | null {
  if (source.token) return source.invitedCampaignId ?? null
  const typed = Number(source.typedId)
  return Number.isInteger(typed) && typed > 0 ? typed : null
}

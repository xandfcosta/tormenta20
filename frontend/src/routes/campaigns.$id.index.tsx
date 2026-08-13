import { createFileRoute } from '@tanstack/solid-router'
import { z } from 'zod'
import { CampaignDetailPage } from '@/pages/campaigns/campaign-detail-page'
import { requireSession } from './-guards'

// The active section lives in the URL so it deep-links and survives the back
// button — and, unlike the React version, it is the ONLY place it lives.
const searchSchema = z.object({ tab: z.string().optional() })

export const Route = createFileRoute('/campaigns/$id/')({
  validateSearch: searchSchema,
  beforeLoad: requireSession,
  component: CampaignDetailPage,
})

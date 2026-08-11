import { createFileRoute } from '@tanstack/solid-router'
import { NewCampaignPage } from '@/pages/campaigns/campaign-new-page'
import { requireSession } from './-guards'

export const Route = createFileRoute('/campaigns/new')({
  beforeLoad: requireSession,
  component: NewCampaignPage,
})

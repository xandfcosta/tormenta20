import { createFileRoute } from '@tanstack/react-router'
import { CampaignDetailPage } from '@/pages/campaigns/campaign-detail-page'

// Campaign detail at exactly `/campaigns/$id`. Data is prefetched by the
// parent layout route; this just renders the screen.
export const Route = createFileRoute('/campaigns/$id/')({
  component: CampaignDetailPage,
})

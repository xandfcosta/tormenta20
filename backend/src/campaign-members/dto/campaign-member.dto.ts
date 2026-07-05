import { IsIn, IsInt, IsOptional, Min } from 'class-validator';

export const CAMPAIGN_MEMBER_ROLES = ['player', 'gm'] as const;
export type CampaignMemberRole = (typeof CAMPAIGN_MEMBER_ROLES)[number];

export class AddMemberDto {
  @IsInt()
  @Min(1)
  characterId!: number;

  @IsOptional()
  @IsIn(CAMPAIGN_MEMBER_ROLES)
  role?: CampaignMemberRole;
}

export class UpdateMemberDto {
  @IsIn(CAMPAIGN_MEMBER_ROLES)
  role!: CampaignMemberRole;
}

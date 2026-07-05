import {
  BadRequestException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { CreateCampaignDto, UpdateCampaignDto } from './dto/campaign.dto';

@Injectable()
export class CampaignsService {
  constructor(private readonly prisma: PrismaService) {}

  list(ownerId: number) {
    return this.prisma.campaign.findMany({
      where: { ownerId },
      orderBy: { updatedAt: 'desc' },
    });
  }

  async findOne(ownerId: number, id: number) {
    const campaign = await this.prisma.campaign.findUnique({ where: { id } });
    if (!campaign) throw new NotFoundException(`Campaign ${id} not found`);
    if (campaign.ownerId !== ownerId) {
      throw new ForbiddenException(`Campaign ${id} belongs to another user`);
    }
    return campaign;
  }

  create(ownerId: number, dto: CreateCampaignDto) {
    return this.prisma.campaign.create({
      data: {
        ownerId,
        name: dto.name.trim(),
        description: dto.description?.trim() ?? null,
      },
    });
  }

  async update(ownerId: number, id: number, dto: UpdateCampaignDto) {
    await this.findOne(ownerId, id);
    const data: { name?: string; description?: string | null } = {};
    if (dto.name !== undefined) data.name = dto.name.trim();
    if (dto.description !== undefined) {
      data.description = dto.description.trim() || null;
    }
    if (Object.keys(data).length === 0) {
      throw new BadRequestException('No fields to update');
    }
    return this.prisma.campaign.update({ where: { id }, data });
  }

  async remove(ownerId: number, id: number) {
    await this.findOne(ownerId, id);
    await this.prisma.campaign.delete({ where: { id } });
    return { id };
  }
}

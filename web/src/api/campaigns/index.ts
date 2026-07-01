import { defHttp } from '@/utils/http';

enum Api {
  Campaigns = '/campaigns',
}

export function getCampaigns(): Promise<any[]> {
  return defHttp.get({ url: `${Api.Campaigns}/` });
}

export function getCampaignSummaries(): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/summary` });
}

export function getCampaign(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}` });
}

export function createCampaign(data: any): Promise<any> {
  return defHttp.post({ url: `${Api.Campaigns}/`, data });
}

export function deleteCampaign(id: number): Promise<void> {
  return defHttp.delete({ url: `${Api.Campaigns}/${id}` });
}

export function completeCampaign(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}/complete` });
}

export function launchCampaign(id: number): Promise<void> {
  return defHttp.post({ url: `${Api.Campaigns}/${id}/launch` });
}

export function getCampaignSummary(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}/summary` });
}

export function getCampaignResults(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}/results` });
}

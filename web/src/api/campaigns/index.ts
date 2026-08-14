import { defHttp } from "@/utils/http";

enum Api {
  Campaigns = "/campaigns",
}

export function getCampaigns(params?: {
  pageNum?: number;
  pageSize?: number;
}): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/`, params });
}

export function getCampaignSummaries(params?: {
  pageNum?: number;
  pageSize?: number;
}): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/summary`, params });
}

export function getDashboardStats(): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/dashboard-stats` });
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

export function getCampaignResults(
  id: number,
  params?: { pageNum?: number; pageSize?: number },
): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}/results`, params });
}

export function getClientCode(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}/client/code` });
}

export function getPageURL(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}/page/url` });
}

export function getCampaignReports(
  id: number,
  params?: { pageNum?: number; pageSize?: number },
): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}/reports`, params });
}

export function getCampaignReportSummary(
  id: number,
  params?: { pageNum?: number; pageSize?: number },
): Promise<any> {
  return defHttp.get({ url: `${Api.Campaigns}/${id}/reports/summary`, params });
}

export function exportCampaignReports(id: number): Promise<void> {
  return defHttp.get({
    url: `${Api.Campaigns}/${id}/reports/export`,
    responseType: "blob",
  });
}

export function exportCampaignResults(id: number): Promise<void> {
  return defHttp.get({
    url: `${Api.Campaigns}/${id}/results/export`,
    responseType: "blob",
  });
}

export function exportCampaignEvents(id: number): Promise<void> {
  return defHttp.get({
    url: `${Api.Campaigns}/${id}/events/export`,
    responseType: "blob",
  });
}

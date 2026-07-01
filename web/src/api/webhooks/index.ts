import { defHttp } from '@/utils/http';

enum Api {
  Webhooks = '/webhooks',
}

export function getWebhooks(): Promise<any[]> {
  return defHttp.get({ url: `${Api.Webhooks}/` });
}

export function getWebhook(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Webhooks}/${id}` });
}

export function createWebhook(data: any): Promise<any> {
  return defHttp.post({ url: `${Api.Webhooks}/`, data });
}

export function updateWebhook(id: number, data: any): Promise<any> {
  return defHttp.put({ url: `${Api.Webhooks}/${id}`, data: { ...data, id } });
}

export function deleteWebhook(id: number): Promise<void> {
  return defHttp.delete({ url: `${Api.Webhooks}/${id}` });
}

export function validateWebhook(id: number): Promise<void> {
  return defHttp.post({ url: `${Api.Webhooks}/${id}/validate` });
}

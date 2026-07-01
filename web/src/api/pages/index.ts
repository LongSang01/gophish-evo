import { defHttp } from '@/utils/http';

enum Api {
  Pages = '/pages',
}

export function getPages(): Promise<any[]> {
  return defHttp.get({ url: `${Api.Pages}/` });
}

export function getPage(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Pages}/${id}` });
}

export function createPage(data: any): Promise<any> {
  return defHttp.post({ url: `${Api.Pages}/`, data });
}

export function updatePage(id: number, data: any): Promise<any> {
  return defHttp.put({ url: `${Api.Pages}/${id}`, data: { ...data, id } });
}

export function deletePage(id: number): Promise<void> {
  return defHttp.delete({ url: `${Api.Pages}/${id}` });
}

export function importSite(data: { url: string; include_resources?: boolean }): Promise<any> {
  return defHttp.post({ url: '/import/site', data });
}

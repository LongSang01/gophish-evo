import { defHttp } from '@/utils/http';

enum Api {
  Templates = '/templates',
}

export function getTemplates(): Promise<any[]> {
  return defHttp.get({ url: `${Api.Templates}/` });
}

export function getTemplate(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Templates}/${id}` });
}

export function createTemplate(data: any): Promise<any> {
  return defHttp.post({ url: `${Api.Templates}/`, data });
}

export function updateTemplate(id: number, data: any): Promise<any> {
  return defHttp.put({ url: `${Api.Templates}/${id}`, data: { ...data, id } });
}

export function deleteTemplate(id: number): Promise<void> {
  return defHttp.delete({ url: `${Api.Templates}/${id}` });
}

export function importEmail(data: { content: string; convert_links: boolean }): Promise<any> {
  return defHttp.post({ url: '/import/email', data });
}

export function sendTestEmail(data: {
  template?: { name: string };
  page?: { name: string };
  smtp: { name: string };
  email: string;
  full_name?: string;
  position?: string;
}): Promise<any> {
  return defHttp.post({ url: '/util/send_test_email', data });
}

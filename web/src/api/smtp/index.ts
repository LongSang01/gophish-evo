import { defHttp } from '@/utils/http';

enum Api {
  SMTP = '/smtp',
}

export function getSMTPProfiles(): Promise<any[]> {
  return defHttp.get({ url: `${Api.SMTP}/` });
}

export function getSMTPProfile(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.SMTP}/${id}` });
}

export function createSMTPProfile(data: any): Promise<any> {
  return defHttp.post({ url: `${Api.SMTP}/`, data });
}

export function updateSMTPProfile(id: number, data: any): Promise<any> {
  return defHttp.put({ url: `${Api.SMTP}/${id}`, data: { ...data, id } });
}

export function deleteSMTPProfile(id: number): Promise<void> {
  return defHttp.delete({ url: `${Api.SMTP}/${id}` });
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

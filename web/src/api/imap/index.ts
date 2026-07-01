import { defHttp } from '@/utils/http';

export function getIMAPSettings(): Promise<any> {
  return defHttp.get({ url: '/imap/' });
}

export function saveIMAPSettings(data: any): Promise<any> {
  return defHttp.post({ url: '/imap/', data });
}

export function validateIMAP(data: any): Promise<any> {
  return defHttp.post({ url: '/imap/validate', data });
}

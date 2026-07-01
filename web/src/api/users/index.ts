import { defHttp } from '@/utils/http';

enum Api {
  Users = '/users',
}

export function getUsers(): Promise<any[]> {
  return defHttp.get({ url: `${Api.Users}/` });
}

export function getUser(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Users}/${id}` });
}

export function createUser(data: any): Promise<any> {
  return defHttp.post({ url: `${Api.Users}/`, data });
}

export function updateUser(id: number, data: any): Promise<any> {
  return defHttp.put({ url: `${Api.Users}/${id}`, data: { ...data, id } });
}

export function deleteUser(id: number): Promise<void> {
  return defHttp.delete({ url: `${Api.Users}/${id}` });
}

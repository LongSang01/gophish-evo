import { defHttp } from '@/utils/http';

enum Api {
  Groups = '/groups',
}

export function getGroups(): Promise<any[]> {
  return defHttp.get({ url: `${Api.Groups}/` });
}

export function getGroupsSummary(): Promise<any> {
  return defHttp.get({ url: `${Api.Groups}/summary` });
}

export function getGroup(id: number): Promise<any> {
  return defHttp.get({ url: `${Api.Groups}/${id}` });
}

export function createGroup(data: any): Promise<any> {
  return defHttp.post({ url: `${Api.Groups}/`, data });
}

export function updateGroup(id: number, data: any): Promise<any> {
  return defHttp.put({ url: `${Api.Groups}/${id}`, data: { ...data, id } });
}

export function deleteGroup(id: number): Promise<void> {
  return defHttp.delete({ url: `${Api.Groups}/${id}` });
}

export function importGroup(groupId: number, file: File): Promise<any> {
  const formData = new FormData();
  formData.append('file', file);
  return defHttp.post({
    url: `/import/group?group_id=${groupId}`,
    data: formData,
    headers: { 'Content-Type': 'multipart/form-data' },
  });
}

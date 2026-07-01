import { defHttp } from '@/utils/http';
import type { LoginParams, LoginResult, UserInfo, ChangePasswordParams } from './types';

enum Api {
  Login = '/auth/login',
  Logout = '/auth/logout',
  CurrentUser = '/auth/me',
  ChangePassword = '/auth/change-password',
}

export function login(params: LoginParams): Promise<LoginResult> {
  return defHttp.post({ url: Api.Login, data: params });
}

export function logout(): Promise<void> {
  return defHttp.post({ url: Api.Logout });
}

export function getCurrentUser(): Promise<UserInfo> {
  return defHttp.get({ url: Api.CurrentUser });
}

export function changePassword(params: ChangePasswordParams): Promise<void> {
  return defHttp.post({ url: Api.ChangePassword, data: params });
}

export function resetApiKey(): Promise<any> {
  return defHttp.post({ url: '/reset' });
}

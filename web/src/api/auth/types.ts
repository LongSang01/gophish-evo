export interface LoginParams {
  username: string;
  password: string;
}

export interface LoginResult {
  success: boolean;
  message: string;
  token: string;
  user: UserInfo;
}

export interface UserInfo {
  id: number;
  username: string;
  api_key: string;
  role: {
    id: number;
    name: string;
    slug: string;
  };
  password_change_required: boolean;
  account_locked: boolean;
  last_login: string;
}

export interface ChangePasswordParams {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

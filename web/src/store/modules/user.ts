import { defineStore } from 'pinia';
import { ref } from 'vue';
import { login, logout, getCurrentUser } from '@/api/auth';
import type { UserInfo, LoginParams } from '@/api/auth/types';

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem('gophish_evo_token') || '');
  const userInfo = ref<UserInfo | null>(null);

  // Set token
  function setToken(newToken: string) {
    token.value = newToken;
    localStorage.setItem('gophish_evo_token', newToken);
  }

  // Clear token
  function clearToken() {
    token.value = '';
    userInfo.value = null;
    localStorage.removeItem('gophish_evo_token');
  }

  // Login
  async function loginAction(params: LoginParams) {
    const result = await login(params);
    if (result.token) {
      setToken(result.token);
      userInfo.value = result.user;
    }
    return result;
  }

  // Logout
  async function logoutAction() {
    try {
      await logout();
    } finally {
      clearToken();
    }
  }

  // Get user info
  async function getUserInfo() {
    const result = await getCurrentUser();
    userInfo.value = result;
    return result;
  }

  return {
    token,
    userInfo,
    setToken,
    clearToken,
    loginAction,
    logoutAction,
    getUserInfo,
  };
});

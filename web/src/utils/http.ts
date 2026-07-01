import axios from 'axios';
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { message } from 'ant-design-vue';
import { useUserStore } from '@/store/modules/user';

// Create axios instance
const service: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor
service.interceptors.request.use(
  (config) => {
    const userStore = useUserStore();
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor
service.interceptors.response.use(
  (response: AxiosResponse) => {
    return response.data;
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response;
      switch (status) {
        case 401:
          // Don't redirect if already on login page (login failure)
          if (!window.location.pathname.startsWith('/login')) {
            message.error('登录已过期，请重新登录');
            const userStore = useUserStore();
            userStore.clearToken();
            window.location.href = '/login';
          }
          break;
        case 403:
          message.error('没有权限访问');
          break;
        case 404:
          message.error('请求的资源不存在');
          break;
        case 500:
          message.error('服务器错误');
          break;
        default:
          message.error(data?.message || '请求失败');
      }
    } else {
      message.error('网络错误');
    }
    return Promise.reject(error);
  }
);

// HTTP methods
export const defHttp = {
  get: <T = any>(config: AxiosRequestConfig): Promise<T> => {
    return service({ ...config, method: 'GET' });
  },
  post: <T = any>(config: AxiosRequestConfig): Promise<T> => {
    return service({ ...config, method: 'POST' });
  },
  put: <T = any>(config: AxiosRequestConfig): Promise<T> => {
    return service({ ...config, method: 'PUT' });
  },
  delete: <T = any>(config: AxiosRequestConfig): Promise<T> => {
    return service({ ...config, method: 'DELETE' });
  },
};

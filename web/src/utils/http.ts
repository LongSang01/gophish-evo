import axios from "axios";
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from "axios";
import { message } from "ant-design-vue";
import { useUserStore } from "@/store/modules/user";

// Create axios instance
const service: AxiosInstance = axios.create({
  baseURL: "/api",
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
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
  },
);

// Response interceptor
service.interceptors.response.use(
  (response: AxiosResponse) => {
    // Blob responses (CSV export etc.) must be returned as-is
    if (response.config.responseType === "blob") {
      return response.data;
    }
    const body = response.data;
    // If the response has a success field and it's false, treat as error
    if (
      body &&
      typeof body === "object" &&
      "success" in body &&
      !body.success
    ) {
      return Promise.reject(new Error(body.message || "请求失败"));
    }
    // List response: {success, items, total}
    if (body && typeof body === "object" && "items" in body) {
      return body;
    }
    // Single object response: {success, data}
    if (
      body &&
      typeof body === "object" &&
      "data" in body &&
      "success" in body
    ) {
      return body.data;
    }
    // Fallback: return raw body (login, logout, action responses, etc.)
    return body;
  },
  async (error) => {
    if (error.response) {
      let { status, data } = error.response;
      // When responseType is "blob", error data is a Blob — read it to extract
      // the JSON error message so users see a meaningful error instead of a
      // broken file download.
      if (data instanceof Blob) {
        try {
          const text = await data.text();
          const parsed = JSON.parse(text);
          if (parsed && typeof parsed === "object") {
            data = parsed;
          }
        } catch {
          // not JSON — ignore
        }
      }
      switch (status) {
        case 401:
          if (window.location.pathname.startsWith("/login")) {
            // Login failure: show the server error message directly
            message.error(data?.message || "请求失败");
          } else {
            message.error("登录已过期，请重新登录");
            const userStore = useUserStore();
            userStore.clearToken();
            window.location.href = "/login";
          }
          return Promise.reject(new Error(data?.message || "请求失败"));
        case 403:
          message.error("没有权限访问");
          break;
        case 404:
          message.error("请求的资源不存在");
          break;
        case 500:
          message.error("服务器错误");
          break;
        default:
          message.error(data?.message || "请求失败");
      }
    } else {
      message.error("网络错误");
    }
    return Promise.reject(error);
  },
);

// HTTP methods
export const defHttp = {
  get: <T = any>(config: AxiosRequestConfig): Promise<T> => {
    return service({ ...config, method: "GET" });
  },
  post: <T = any>(config: AxiosRequestConfig): Promise<T> => {
    return service({ ...config, method: "POST" });
  },
  put: <T = any>(config: AxiosRequestConfig): Promise<T> => {
    return service({ ...config, method: "PUT" });
  },
  delete: <T = any>(config: AxiosRequestConfig): Promise<T> => {
    return service({ ...config, method: "DELETE" });
  },
};

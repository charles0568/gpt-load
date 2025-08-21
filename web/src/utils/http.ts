import { useAuthService } from "@/services/auth";
import axios from "axios";
import { appState } from "./app-state";

// 定义不需要显示 loading 的 API 地址列表
const noLoadingUrls = ["/tasks/status"];

declare module "axios" {
  interface AxiosRequestConfig {
    hideMessage?: boolean;
  }
}

const http = axios.create({
  baseURL: "/api",
  timeout: 60000,
  headers: { "Content-Type": "application/json" },
});

// 请求拦截器
http.interceptors.request.use(config => {
  // 检查当前请求的 URL 是否在屏蔽列表中
  if (config.url && !noLoadingUrls.includes(config.url)) {
    appState.loading = true;
  }
  const authKey = localStorage.getItem("authKey");
  if (authKey) {
    config.headers.Authorization = `Bearer ${authKey}`;
  }
  return config;
});

// 回應攔截器
http.interceptors.response.use(
  response => {
    appState.loading = false;
    if (response.config.method !== "get" && !response.config.hideMessage) {
      window.$message.success(response.data.message ?? "操作成功");
    }
    return response.data;
  },
  error => {
    appState.loading = false;
    if (error.response) {
      if (error.response.status === 401) {
        if (window.location.pathname !== "/login") {
          const { logout } = useAuthService();
          logout();
          window.location.href = "/login";
        }
      }
      window.$message.error(error.response.data?.message || `請求失敗: ${error.response.status}`, {
        keepAliveOnHover: true,
        duration: 5000,
        closable: true,
      });
    } else if (error.request) {
      window.$message.error("網路錯誤，請檢查您的連線");
    } else {
      window.$message.error("請求設定錯誤");
    }
    return Promise.reject(error);
  }
);

export default http;

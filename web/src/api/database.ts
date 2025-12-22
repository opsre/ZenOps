import axios, { AxiosError } from 'axios';
import { MessagePlugin } from 'tdesign-vue-next';

// 后端统一响应格式
interface ApiResponse<T = any> {
  code: number;
  message: string;
  data?: T;
}

// 请求日志
export interface RequestLog {
  id: number;
  request_id: string;
  source: string;
  method: string;
  path: string;
  user_id?: string;
  user_name?: string;
  request_body?: string;
  response_body?: string;
  status_code: number;
  duration: number;
  ip?: string;
  user_agent?: string;
  error?: string;
  remark?: string;
  created_at: string;
  updated_at: string;
}

// 请求日志列表响应
export interface RequestLogListResponse {
  total: number;
  page: number;
  page_size: number;
  data: RequestLog[];
}

// 请求日志统计
export interface RequestLogStats {
  total_count: number;
  avg_duration: number;
  source_stats: Array<{ source: string; count: number }>;
  status_stats: Array<{ status_code: number; count: number }>;
  start_time: string;
  end_time: string;
}

// 配置快照
export interface ConfigSnapshot {
  id: number;
  version: string;
  config_type: string;
  config_key: string;
  config_value: string;
  description?: string;
  operator?: string;
  remark?: string;
  created_at: string;
  updated_at: string;
}

// 配置快照列表响应
export interface ConfigSnapshotListResponse {
  total: number;
  page: number;
  page_size: number;
  data: ConfigSnapshot[];
}

const api = axios.create({
  baseURL: '/api/v1/database',
  timeout: 30000,
});

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    const { code, message, data } = response.data as ApiResponse;
    if (code === 200) {
      return data;
    }
    MessagePlugin.error(message || '请求失败');
    return Promise.reject(new Error(message));
  },
  (error: AxiosError<ApiResponse>) => {
    const message = error.response?.data?.message || error.message || '网络错误';
    MessagePlugin.error(message);
    return Promise.reject(error);
  }
);

// ==================== 请求日志 API ====================

export const getRequestLogs = async (params: {
  page?: number;
  page_size?: number;
  source?: string;
  user_id?: string;
  method?: string;
  status_code?: number;
  start_time?: string;
  end_time?: string;
}): Promise<RequestLogListResponse> => {
  return await api.get('/logs/list', { params });
};

export const getRequestLog = async (id: string): Promise<RequestLog> => {
  return await api.get(`/logs/${id}`);
};

export const getRequestLogStats = async (params?: {
  start_time?: string;
  end_time?: string;
}): Promise<RequestLogStats> => {
  return await api.get('/logs/stats', { params });
};

export const cleanupRequestLogs = async (days: number): Promise<{ deleted: number; days: number }> => {
  return await api.post('/logs/cleanup', { days });
};

// ==================== 配置快照 API ====================

export const getConfigSnapshots = async (params: {
  page?: number;
  page_size?: number;
}): Promise<ConfigSnapshotListResponse> => {
  return await api.get('/config/list', { params });
};

export const createConfigSnapshot = async (snapshot: {
  version: string;
  config_type: string;
  config_key: string;
  config_value: string;
  description?: string;
  operator?: string;
  remark?: string;
}): Promise<ConfigSnapshot> => {
  return await api.post('/config/create', snapshot);
};

export const getConfigSnapshotHistory = async (params: {
  config_type: string;
  config_key: string;
  limit?: number;
}): Promise<{ config_type: string; config_key: string; data: ConfigSnapshot[] }> => {
  return await api.get('/config/history', { params });
};

export const getConfigSnapshotByVersion = async (version: string): Promise<{ version: string; data: ConfigSnapshot[] }> => {
  return await api.get(`/config/version/${version}`);
};

import axios, { AxiosError } from 'axios';
import type { MCPServerConfig, MCPTool, MCPToolCallResult } from '@/types/mcp';
import { MessagePlugin } from 'tdesign-vue-next';

// 后端统一响应格式
interface ApiResponse<T = any> {
  code: number;
  message: string;
  data?: T;
}

const api = axios.create({
  baseURL: '/api/v1/mcp',
  timeout: 30000,
});

// 响应拦截器 - 统一处理后端返回的数据
api.interceptors.response.use(
  (response) => {
    const { code, message, data } = response.data as ApiResponse;
    if (code === 200) {
      return data; // 直接返回data部分
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

export const getMCPServers = async (): Promise<MCPServerConfig[]> => {
  const data: any = await api.get('/servers');
  return data?.servers || [];
};

export const getMCPServer = async (name: string): Promise<MCPServerConfig> => {
  const data: any = await api.get(`/servers/${name}`);
  return data.server;
};

export const addMCPServer = async (config: MCPServerConfig): Promise<MCPServerConfig> => {
  const data: any = await api.post('/servers', config);
  return data.server;
};

export const updateMCPServer = async (name: string, config: MCPServerConfig): Promise<MCPServerConfig> => {
  const data: any = await api.put(`/servers/${name}`, config);
  return data.server;
};

export const deleteMCPServer = async (name: string): Promise<void> => {
  await api.delete(`/servers/${name}`);
};

export const toggleMCPServer = async (name: string, isActive: boolean): Promise<MCPServerConfig> => {
  const data: any = await api.patch(`/servers/${name}/toggle`, { isActive });
  return data.server;
};

export const getMCPTools = async (name: string): Promise<MCPTool[]> => {
  const data: any = await api.get(`/servers/${name}/tools`);
  return data?.tools || [];
};

export const testMCPTool = async (serverName: string, toolName: string, args: any): Promise<MCPToolCallResult> => {
  const data: any = await api.post(`/servers/${serverName}/tools/${toolName}/test`, args);
  return data.result;
};

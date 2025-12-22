import axios from 'axios';

const API_BASE_URL = '/api/v1';

// ==================== 类型定义 ====================

// 云服务商配置
export interface ProviderConfig {
  id?: number;
  provider: string; // aliyun, tencent
  account_name: string;
  enabled: boolean;
  access_key_id: string;
  access_key_secret: string;
  regions: string; // JSON数组字符串
  extra?: string; // JSON对象字符串
  created_at?: string;
  updated_at?: string;
}

// 集成应用配置
export interface IntegrationConfig {
  id?: number;
  platform: string; // dingtalk, feishu, wecom
  enabled: boolean;
  app_key?: string; // 钉钉
  app_secret?: string; // 钉钉/飞书
  agent_id?: string; // 钉钉
  card_template_id?: string; // 钉钉
  app_id?: string; // 飞书
  token?: string; // 企业微信
  encoding_aes_key?: string; // 企业微信
  created_at?: string;
  updated_at?: string;
}

// Jenkins配置
export interface JenkinsConfig {
  id?: number;
  enabled: boolean;
  url: string;
  username: string;
  token: string;
  created_at?: string;
  updated_at?: string;
}

// LLM配置
export interface LLMConfig {
  id?: number;
  enabled: boolean;
  model: string;
  api_key: string;
  base_url?: string;
  created_at?: string;
  updated_at?: string;
}

// 服务器配置
export interface ServerConfig {
  id?: number;
  http_enabled: boolean;
  http_port: number;
  mcp_enabled: boolean;
  mcp_port: number;
  auto_register_external_tools: boolean;
  tool_name_format: string;
  created_at?: string;
  updated_at?: string;
}

// ==================== API 函数 ====================

// ==================== 云服务商配置 ====================

export async function getProviderConfigList() {
  const response = await axios.get(`${API_BASE_URL}/config/provider`);
  return response.data;
}

export async function getProviderConfig(id: number) {
  const response = await axios.get(`${API_BASE_URL}/config/provider/${id}`);
  return response.data;
}

export async function createProviderConfig(data: ProviderConfig) {
  const response = await axios.post(`${API_BASE_URL}/config/provider`, data);
  return response.data;
}

export async function updateProviderConfig(id: number, data: ProviderConfig) {
  const response = await axios.put(`${API_BASE_URL}/config/provider/${id}`, data);
  return response.data;
}

export async function deleteProviderConfig(id: number) {
  const response = await axios.delete(`${API_BASE_URL}/config/provider/${id}`);
  return response.data;
}

// ==================== 集成应用配置 ====================

export async function getIntegrationConfigList() {
  const response = await axios.get(`${API_BASE_URL}/config/integration`);
  return response.data;
}

export async function getIntegrationConfig(id: number) {
  const response = await axios.get(`${API_BASE_URL}/config/integration/${id}`);
  return response.data;
}

export async function createIntegrationConfig(data: IntegrationConfig) {
  const response = await axios.post(`${API_BASE_URL}/config/integration`, data);
  return response.data;
}

export async function updateIntegrationConfig(id: number, data: IntegrationConfig) {
  const response = await axios.put(`${API_BASE_URL}/config/integration/${id}`, data);
  return response.data;
}

export async function deleteIntegrationConfig(id: number) {
  const response = await axios.delete(`${API_BASE_URL}/config/integration/${id}`);
  return response.data;
}

// ==================== Jenkins配置 ====================

export async function getJenkinsConfig() {
  const response = await axios.get(`${API_BASE_URL}/config/jenkins`);
  return response.data;
}

export async function saveJenkinsConfig(data: JenkinsConfig) {
  const response = await axios.post(`${API_BASE_URL}/config/jenkins`, data);
  return response.data;
}

// ==================== LLM配置 ====================

export async function getLLMConfig() {
  const response = await axios.get(`${API_BASE_URL}/config/llm`);
  return response.data;
}

export async function saveLLMConfig(data: LLMConfig) {
  const response = await axios.post(`${API_BASE_URL}/config/llm`, data);
  return response.data;
}

// ==================== 服务器配置 ====================

export async function getServerConfig() {
  const response = await axios.get(`${API_BASE_URL}/config/server`);
  return response.data;
}

export async function saveServerConfig(data: ServerConfig) {
  const response = await axios.post(`${API_BASE_URL}/config/server`, data);
  return response.data;
}

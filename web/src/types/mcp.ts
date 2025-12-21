export interface MCPServerConfig {
  isActive: boolean;
  name: string;
  type: 'stdio' | 'sse' | 'streamableHttp';
  description: string;
  baseUrl?: string;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  headers?: Record<string, string>;
  provider: string;
  providerUrl: string;
  logoUrl?: string;
  tags: string[];
  longRunning: boolean;
  timeout: number;
  installSource: string;
  toolPrefix: string;
  autoRegister: boolean;
}

export interface MCPTool {
  name: string;
  description?: string;
  inputSchema?: any;
}

export interface MCPToolCallResult {
  content: Array<{
    type: string;
    text: string;
  }>;
  isError?: boolean;
}

import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { MCPServerConfig, MCPTool } from '@/types/mcp';
import * as mcpApi from '@/api/mcp';
import { MessagePlugin } from 'tdesign-vue-next';

export const useMCPStore = defineStore('mcp', () => {
  const servers = ref<MCPServerConfig[]>([]);
  const currentServer = ref<MCPServerConfig | null>(null);
  const tools = ref<MCPTool[]>([]);
  const loading = ref(false);

  const fetchServers = async () => {
    loading.value = true;
    try {
      servers.value = await mcpApi.getMCPServers();
    } catch (error) {
      console.error('Failed to fetch MCP servers:', error);
      MessagePlugin.error('获取 MCP 服务列表失败');
    } finally {
      loading.value = false;
    }
  };

  const fetchServer = async (name: string) => {
    loading.value = true;
    try {
      currentServer.value = await mcpApi.getMCPServer(name);
    } catch (error) {
      console.error(`Failed to fetch MCP server ${name}:`, error);
      MessagePlugin.error('获取 MCP 服务详情失败');
    } finally {
      loading.value = false;
    }
  };

  const addServer = async (config: MCPServerConfig) => {
    try {
      await mcpApi.addMCPServer(config);
      MessagePlugin.success('添加 MCP 服务成功');
      await fetchServers();
      return true;
    } catch (error) {
      console.error('Failed to add MCP server:', error);
      MessagePlugin.error('添加 MCP 服务失败');
      return false;
    }
  };

  const updateServer = async (name: string, config: MCPServerConfig) => {
    try {
      await mcpApi.updateMCPServer(name, config);
      MessagePlugin.success('更新 MCP 服务成功');
      await fetchServers();
      return true;
    } catch (error) {
      console.error('Failed to update MCP server:', error);
      MessagePlugin.error('更新 MCP 服务失败');
      return false;
    }
  };

  const deleteServer = async (name: string) => {
    try {
      await mcpApi.deleteMCPServer(name);
      MessagePlugin.success('删除 MCP 服务成功');
      await fetchServers();
      return true;
    } catch (error) {
      console.error('Failed to delete MCP server:', error);
      MessagePlugin.error('删除 MCP 服务失败');
      return false;
    }
  };

  const toggleServer = async (name: string, isActive: boolean) => {
    try {
      await mcpApi.toggleMCPServer(name, isActive);
      MessagePlugin.success(`${isActive ? '启用' : '禁用'} MCP 服务成功`);
      await fetchServers();
      return true;
    } catch (error) {
      console.error('Failed to toggle MCP server:', error);
      MessagePlugin.error(`${isActive ? '启用' : '禁用'} MCP 服务失败`);
      return false;
    }
  };

  const fetchTools = async (name: string) => {
    loading.value = true;
    try {
      tools.value = await mcpApi.getMCPTools(name);
    } catch (error) {
      console.error(`Failed to fetch tools for ${name}:`, error);
      MessagePlugin.error('获取工具列表失败');
      tools.value = [];
    } finally {
      loading.value = false;
    }
  };

  const testTool = async (serverName: string, toolName: string, args: any) => {
    try {
      return await mcpApi.testMCPTool(serverName, toolName, args);
    } catch (error) {
      console.error('Failed to test tool:', error);
      MessagePlugin.error('工具调用失败');
      throw error;
    }
  };

  return {
    servers,
    currentServer,
    tools,
    loading,
    fetchServers,
    fetchServer,
    addServer,
    updateServer,
    deleteServer,
    toggleServer,
    fetchTools,
    testTool,
  };
});

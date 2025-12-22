import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import MCPList from '@/views/mcp/MCPList.vue';
import RequestLogs from '@/views/database/RequestLogs.vue';
import ConfigSnapshots from '@/views/database/ConfigSnapshots.vue';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/mcp',
  },
  {
    path: '/mcp',
    name: 'MCPList',
    component: MCPList,
    meta: {
      title: 'MCP Server 管理',
    },
  },
  {
    path: '/database/logs',
    name: 'RequestLogs',
    component: RequestLogs,
    meta: {
      title: '请求日志',
    },
  },
  {
    path: '/database/config',
    name: 'ConfigSnapshots',
    component: ConfigSnapshots,
    meta: {
      title: '配置快照',
    },
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;

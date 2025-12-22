import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import MCPList from '@/views/mcp/MCPList.vue';
import ConfigManagement from '@/views/config/ConfigManagement.vue';

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
    path: '/config',
    name: 'ConfigManagement',
    component: ConfigManagement,
    meta: {
      title: '配置管理',
    },
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;

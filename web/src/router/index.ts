import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import MCPList from '@/views/mcp/MCPList.vue';

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
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;

<template>
  <t-layout class="app-shell">
    <t-header class="app-header">
      <div class="nav-bar">
        <div class="brand">
          <img class="brand-logo" src="/favicon.svg" alt="ZenOps Logo" />
          <div class="brand-text">
            <h1 class="brand-title">ZenOps</h1>
            <p class="brand-subtitle">智能运维平台</p>
          </div>
        </div>
        <div class="nav-links">
          <router-link to="/mcp" class="link" :class="{ active: $route.path === '/mcp' }">MCP 管理</router-link>
          <router-link to="/database/logs" class="link" :class="{ active: $route.path.startsWith('/database/logs') }">请求日志</router-link>
          <router-link to="/database/config" class="link" :class="{ active: $route.path.startsWith('/database/config') }">配置快照</router-link>
          <router-link to="/about" class="link disabled">关于</router-link>
        </div>
        <div class="nav-actions">
          <t-button variant="text" shape="square" size="small">
            <template #icon><t-icon name="search" /></template>
          </t-button>
          <t-button variant="text" shape="square" size="small">
            <template #icon><t-icon name="notification" /></template>
          </t-button>
          <t-avatar size="32px">Z</t-avatar>
        </div>
      </div>
    </t-header>

    <t-content class="app-content">
      <div class="content-container">
        <router-view />
      </div>
    </t-content>

    <t-footer class="app-footer">
      Copyright © 2024 ZenOps · 智能化运维助手
    </t-footer>
  </t-layout>
</template>

<script setup lang="ts">
import { Icon as TIcon } from 'tdesign-icons-vue-next';
</script>

<style scoped>
:global(body) {
  background: #f7f8fb;
}

.app-shell {
  min-height: 100vh;
  background: radial-gradient(circle at 10% 20%, rgba(141, 185, 255, 0.15), transparent 50%),
    radial-gradient(circle at 90% 0%, rgba(248, 171, 210, 0.15), transparent 45%), #f7f8fb;
}

.app-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(16px);
  border-bottom: 1px solid rgba(15, 23, 42, 0.06);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.03);
}

.nav-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 32px;
  padding: 12px clamp(24px, 5vw, 48px);
  max-width: 1400px;
  margin: 0 auto;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-logo {
  width: 36px;
  height: 36px;
  border-radius: 8px;
}

.brand-text {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.brand-title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.brand-subtitle {
  margin: 0;
  font-size: 11px;
  color: #64748b;
  line-height: 1.2;
  font-weight: 500;
}

.nav-links {
  display: flex;
  gap: 8px;
  font-size: 14px;
  flex: 1;
  justify-content: center;
}

.link {
  color: #475569;
  text-decoration: none;
  transition: all 0.2s ease;
  padding: 6px 16px;
  border-radius: 8px;
  font-weight: 500;
  position: relative;
}

.link.active {
  color: #0052d9;
  background: rgba(0, 82, 217, 0.08);
  font-weight: 600;
}

.link.disabled {
  cursor: not-allowed;
  opacity: 0.4;
  pointer-events: none;
}

.link:not(.disabled):not(.active):hover {
  color: #0f172a;
  background: rgba(15, 23, 42, 0.04);
}

.nav-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.app-content {
  padding: 24px 0 48px;
  min-height: calc(100vh - 120px);
}

.content-container {
  width: min(1400px, 100%);
  margin: 0 auto;
  padding: 0 clamp(24px, 5vw, 48px);
}

.app-footer {
  text-align: center;
  color: #94a3b8;
  padding: 20px 0 24px;
  font-size: 12px;
  border-top: 1px solid rgba(15, 23, 42, 0.06);
  background: rgba(255, 255, 255, 0.5);
}

@media (max-width: 960px) {
  .nav-bar {
    flex-wrap: wrap;
    gap: 16px;
    padding: 12px 20px;
  }

  .nav-links {
    order: 3;
    width: 100%;
    justify-content: flex-start;
    gap: 4px;
  }

  .brand-subtitle {
    display: none;
  }
}
</style>

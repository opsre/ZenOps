<template>
  <div class="config-management-page">
    <t-card title="配置管理" :bordered="false">
      <template #subtitle>
        <span class="subtitle-text">管理各类系统配置</span>
      </template>

      <t-tabs v-model="activeTab" @change="handleTabChange">
        <!-- 云服务商配置 -->
        <t-tab-panel value="provider" label="云服务商">
          <ProviderConfig />
        </t-tab-panel>

        <!-- 集成应用配置 -->
        <t-tab-panel value="integration" label="集成应用">
          <IntegrationConfig />
        </t-tab-panel>

        <!-- Jenkins配置 -->
        <t-tab-panel value="jenkins" label="Jenkins">
          <JenkinsConfig />
        </t-tab-panel>

        <!-- LLM配置 -->
        <t-tab-panel value="llm" label="LLM大模型">
          <LLMConfig />
        </t-tab-panel>

        <!-- 服务器配置 -->
        <t-tab-panel value="server" label="服务器">
          <ServerConfig />
        </t-tab-panel>
      </t-tabs>
    </t-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import ProviderConfig from './config/ProviderConfig.vue';
import IntegrationConfig from './config/IntegrationConfig.vue';
import JenkinsConfig from './config/JenkinsConfig.vue';
import LLMConfig from './config/LLMConfig.vue';
import ServerConfig from './config/ServerConfig.vue';

const activeTab = ref('provider');

const handleTabChange = (value: string) => {
  activeTab.value = value;
};
</script>

<style scoped lang="less">
.config-management-page {
  :deep(.t-card__header) {
    padding-bottom: 0;
  }

  .subtitle-text {
    font-size: 13px;
    color: #64748b;
    font-weight: normal;
  }

  :deep(.t-tabs) {
    margin-top: 8px;
  }

  :deep(.t-tab-panel) {
    padding-top: 20px;
  }
}
</style>

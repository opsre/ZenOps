<template>
  <div class="server-config">
    <div class="section">
      <div class="section-header">
        <h3>服务器配置</h3>
        <t-button theme="primary" size="small" @click="showEditDialog">
          <template #icon><t-icon name="edit" /></template>
          编辑配置
        </t-button>
      </div>

      <t-card :bordered="true" v-if="config">
        <t-descriptions :data="configData" :column="2" bordered />
      </t-card>
      <t-card :bordered="true" v-else>
        <t-empty description="暂无配置信息" />
      </t-card>
    </div>

    <!-- 编辑对话框 -->
    <t-dialog
      v-model:visible="formVisible"
      header="编辑服务器配置"
      width="600px"
      :on-confirm="handleSubmit"
      :confirm-btn="{ content: '保存', theme: 'primary', loading: submitting }"
    >
      <t-form :data="formData" ref="formRef" :rules="formRules" label-width="180px">
        <t-form-item label="启用HTTP服务" name="http_enabled">
          <t-switch v-model="formData.http_enabled" />
        </t-form-item>

        <t-form-item label="HTTP端口" name="http_port">
          <t-input-number v-model="formData.http_port" :min="1" :max="65535" />
        </t-form-item>

        <t-form-item label="启用MCP服务" name="mcp_enabled">
          <t-switch v-model="formData.mcp_enabled" />
        </t-form-item>

        <t-form-item label="MCP端口" name="mcp_port">
          <t-input-number v-model="formData.mcp_port" :min="1" :max="65535" />
        </t-form-item>

        <t-form-item label="自动注册外部工具" name="auto_register_external_tools">
          <t-switch v-model="formData.auto_register_external_tools" />
        </t-form-item>

        <t-form-item label="工具名称格式" name="tool_name_format">
          <t-input v-model="formData.tool_name_format" placeholder="例如: {server}_{tool}" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { getServerConfig, saveServerConfig, type ServerConfig } from '@/api/config';
import { MessagePlugin } from 'tdesign-vue-next';

const loading = ref(false);
const submitting = ref(false);
const config = ref<ServerConfig | null>(null);
const formVisible = ref(false);
const formRef = ref();

const formData = reactive<Partial<ServerConfig>>({
  http_enabled: true,
  http_port: 8080,
  mcp_enabled: true,
  mcp_port: 3000,
  auto_register_external_tools: true,
  tool_name_format: '{server}_{tool}',
});

const formRules = {
  http_port: [{ required: true, message: '请输入HTTP端口' }],
  mcp_port: [{ required: true, message: '请输入MCP端口' }],
  tool_name_format: [{ required: true, message: '请输入工具名称格式' }],
};

const configData = computed(() => {
  if (!config.value) return [];

  return [
    { label: 'HTTP服务', value: config.value.http_enabled ? '启用' : '禁用' },
    { label: 'HTTP端口', value: config.value.http_port },
    { label: 'MCP服务', value: config.value.mcp_enabled ? '启用' : '禁用' },
    { label: 'MCP端口', value: config.value.mcp_port },
    { label: '自动注册外部工具', value: config.value.auto_register_external_tools ? '是' : '否' },
    { label: '工具名称格式', value: config.value.tool_name_format },
  ];
});

const fetchConfig = async () => {
  loading.value = true;
  try {
    const result = await getServerConfig();
    config.value = result.data;
  } catch (error) {
    console.error('Failed to fetch config:', error);
  } finally {
    loading.value = false;
  }
};

const showEditDialog = () => {
  if (config.value) {
    Object.assign(formData, {
      http_enabled: config.value.http_enabled,
      http_port: config.value.http_port,
      mcp_enabled: config.value.mcp_enabled,
      mcp_port: config.value.mcp_port,
      auto_register_external_tools: config.value.auto_register_external_tools,
      tool_name_format: config.value.tool_name_format,
    });
  }
  formVisible.value = true;
};

const handleSubmit = async () => {
  const valid = await formRef.value?.validate();
  if (!valid) return;

  submitting.value = true;
  try {
    await saveServerConfig(formData as ServerConfig);
    MessagePlugin.success('保存成功');
    formVisible.value = false;
    fetchConfig();
  } catch (error) {
    console.error('Failed to save config:', error);
    MessagePlugin.error('保存失败');
  } finally {
    submitting.value = false;
  }
};

onMounted(() => {
  fetchConfig();
});
</script>

<style scoped lang="less">
.server-config {
  .section {
    margin-bottom: 24px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: #0f172a;
    }
  }
}
</style>

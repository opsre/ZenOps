<template>
  <div class="llm-config">
    <div class="section">
      <div class="section-header">
        <h3>LLM大模型配置</h3>
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
      header="编辑LLM配置"
      width="600px"
      :on-confirm="handleSubmit"
      :confirm-btn="{ content: '保存', theme: 'primary', loading: submitting }"
    >
      <t-form :data="formData" ref="formRef" :rules="formRules" label-width="120px">
        <t-form-item label="启用状态" name="enabled">
          <t-switch v-model="formData.enabled" />
        </t-form-item>

        <t-form-item label="模型" name="model">
          <t-input v-model="formData.model" placeholder="例如: gpt-4, claude-3-opus" />
        </t-form-item>

        <t-form-item label="API Key" name="api_key">
          <t-input v-model="formData.api_key" type="password" placeholder="请输入API Key" />
        </t-form-item>

        <t-form-item label="Base URL" name="base_url">
          <t-input v-model="formData.base_url" placeholder="可选,自定义API端点" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { getLLMConfig, saveLLMConfig, type LLMConfig } from '@/api/config';
import { MessagePlugin } from 'tdesign-vue-next';

const loading = ref(false);
const submitting = ref(false);
const config = ref<LLMConfig | null>(null);
const formVisible = ref(false);
const formRef = ref();

const formData = reactive<Partial<LLMConfig>>({
  enabled: true,
  model: '',
  api_key: '',
  base_url: '',
});

const formRules = {
  model: [{ required: true, message: '请输入模型名称' }],
  api_key: [{ required: true, message: '请输入API Key' }],
};

const configData = computed(() => {
  if (!config.value) return [];

  return [
    { label: '启用状态', value: config.value.enabled ? '启用' : '禁用' },
    { label: '模型', value: config.value.model },
    { label: 'API Key', value: '******' },
    { label: 'Base URL', value: config.value.base_url || '-' },
  ];
});

const fetchConfig = async () => {
  loading.value = true;
  try {
    const result = await getLLMConfig();
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
      enabled: config.value.enabled,
      model: config.value.model,
      api_key: '',
      base_url: config.value.base_url || '',
    });
  }
  formVisible.value = true;
};

const handleSubmit = async () => {
  const valid = await formRef.value?.validate();
  if (!valid) return;

  submitting.value = true;
  try {
    await saveLLMConfig(formData as LLMConfig);
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
.llm-config {
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

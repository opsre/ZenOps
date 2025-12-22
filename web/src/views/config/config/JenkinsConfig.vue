<template>
  <div class="jenkins-config">
    <div class="section">
      <div class="section-header">
        <h3>Jenkins配置</h3>
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
      header="编辑Jenkins配置"
      width="600px"
      :on-confirm="handleSubmit"
      :confirm-btn="{ content: '保存', theme: 'primary', loading: submitting }"
    >
      <t-form :data="formData" ref="formRef" :rules="formRules" label-width="120px">
        <t-form-item label="启用状态" name="enabled">
          <t-switch v-model="formData.enabled" />
        </t-form-item>

        <t-form-item label="URL" name="url">
          <t-input v-model="formData.url" placeholder="https://jenkins.example.com" />
        </t-form-item>

        <t-form-item label="用户名" name="username">
          <t-input v-model="formData.username" placeholder="请输入用户名" />
        </t-form-item>

        <t-form-item label="Token" name="token">
          <t-input v-model="formData.token" type="password" placeholder="请输入Token" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { getJenkinsConfig, saveJenkinsConfig, type JenkinsConfig } from '@/api/config';
import { MessagePlugin } from 'tdesign-vue-next';

const loading = ref(false);
const submitting = ref(false);
const config = ref<JenkinsConfig | null>(null);
const formVisible = ref(false);
const formRef = ref();

const formData = reactive<Partial<JenkinsConfig>>({
  enabled: true,
  url: '',
  username: '',
  token: '',
});

const formRules = {
  url: [{ required: true, message: '请输入URL' }],
  username: [{ required: true, message: '请输入用户名' }],
  token: [{ required: true, message: '请输入Token' }],
};

const configData = computed(() => {
  if (!config.value) return [];

  return [
    { label: '启用状态', value: config.value.enabled ? '启用' : '禁用' },
    { label: 'URL', value: config.value.url },
    { label: '用户名', value: config.value.username },
    { label: 'Token', value: '******' },
  ];
});

const fetchConfig = async () => {
  loading.value = true;
  try {
    const result = await getJenkinsConfig();
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
      url: config.value.url,
      username: config.value.username,
      token: '',
    });
  }
  formVisible.value = true;
};

const handleSubmit = async () => {
  const valid = await formRef.value?.validate();
  if (!valid) return;

  submitting.value = true;
  try {
    await saveJenkinsConfig(formData as JenkinsConfig);
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
.jenkins-config {
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

<template>
  <div class="integration-config">
    <div class="section">
      <div class="section-header">
        <h3>集成应用配置</h3>
        <t-button theme="primary" size="small" @click="showCreateDialog">
          <template #icon><t-icon name="add" /></template>
          添加配置
        </t-button>
      </div>

      <t-table
        :data="configList"
        :columns="columns"
        :loading="loading"
        row-key="id"
        stripe
        size="medium"
      >
        <template #enabled="{ row }">
          <t-tag :theme="row.enabled ? 'success' : 'default'">
            {{ row.enabled ? '启用' : '禁用' }}
          </t-tag>
        </template>

        <template #platform="{ row }">
          <span>{{ getPlatformName(row.platform) }}</span>
        </template>

        <template #operation="{ row }">
          <t-button theme="primary" variant="text" size="small" @click="handleEdit(row)">
            编辑
          </t-button>
          <t-popconfirm content="确认删除此配置？" @confirm="handleDelete(row.id)">
            <t-button theme="danger" variant="text" size="small">
              删除
            </t-button>
          </t-popconfirm>
        </template>
      </t-table>
    </div>

    <!-- 创建/编辑对话框 -->
    <t-dialog
      v-model:visible="formVisible"
      :header="isEdit ? '编辑集成应用配置' : '添加集成应用配置'"
      width="700px"
      :on-confirm="handleSubmit"
      :confirm-btn="{ content: '保存', theme: 'primary', loading: submitting }"
    >
      <t-form :data="formData" ref="formRef" :rules="formRules" label-width="120px">
        <t-form-item label="平台" name="platform">
          <t-select v-model="formData.platform" placeholder="请选择平台" :disabled="isEdit">
            <t-option value="dingtalk" label="钉钉"></t-option>
            <t-option value="feishu" label="飞书"></t-option>
            <t-option value="wecom" label="企业微信"></t-option>
          </t-select>
        </t-form-item>

        <t-form-item label="启用状态" name="enabled">
          <t-switch v-model="formData.enabled" />
        </t-form-item>

        <!-- 钉钉字段 -->
        <template v-if="formData.platform === 'dingtalk'">
          <t-form-item label="AppKey" name="app_key">
            <t-input v-model="formData.app_key" placeholder="请输入AppKey" />
          </t-form-item>
          <t-form-item label="AppSecret" name="app_secret">
            <t-input v-model="formData.app_secret" type="password" placeholder="请输入AppSecret" />
          </t-form-item>
          <t-form-item label="AgentID" name="agent_id">
            <t-input v-model="formData.agent_id" placeholder="请输入AgentID" />
          </t-form-item>
        </template>

        <!-- 飞书字段 -->
        <template v-if="formData.platform === 'feishu'">
          <t-form-item label="AppID" name="app_id">
            <t-input v-model="formData.app_id" placeholder="请输入AppID" />
          </t-form-item>
          <t-form-item label="AppSecret" name="app_secret">
            <t-input v-model="formData.app_secret" type="password" placeholder="请输入AppSecret" />
          </t-form-item>
        </template>

        <!-- 企业微信字段 -->
        <template v-if="formData.platform === 'wecom'">
          <t-form-item label="Token" name="token">
            <t-input v-model="formData.token" placeholder="请输入Token" />
          </t-form-item>
          <t-form-item label="EncodingAESKey" name="encoding_aes_key">
            <t-input v-model="formData.encoding_aes_key" placeholder="请输入EncodingAESKey" />
          </t-form-item>
        </template>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import {
  getIntegrationConfigList,
  createIntegrationConfig,
  updateIntegrationConfig,
  deleteIntegrationConfig,
  type IntegrationConfig,
} from '@/api/config';
import { MessagePlugin } from 'tdesign-vue-next';

const loading = ref(false);
const submitting = ref(false);
const configList = ref<IntegrationConfig[]>([]);
const formVisible = ref(false);
const isEdit = ref(false);
const formRef = ref();

const formData = reactive<Partial<IntegrationConfig>>({
  platform: '',
  enabled: true,
  app_key: '',
  app_secret: '',
  agent_id: '',
  app_id: '',
  token: '',
  encoding_aes_key: '',
});

const formRules = {
  platform: [{ required: true, message: '请选择平台' }],
};

const columns = [
  { colKey: 'id', title: 'ID', width: 80 },
  { colKey: 'platform', title: '平台', width: 120, cell: 'platform' },
  { colKey: 'enabled', title: '状态', width: 100, cell: 'enabled' },
  { colKey: 'app_key', title: 'AppKey', ellipsis: true },
  { colKey: 'app_id', title: 'AppID', ellipsis: true },
  { colKey: 'operation', title: '操作', width: 150, cell: 'operation' },
];

const getPlatformName = (platform: string) => {
  const names: Record<string, string> = {
    dingtalk: '钉钉',
    feishu: '飞书',
    wecom: '企业微信',
  };
  return names[platform] || platform;
};

const fetchConfigList = async () => {
  loading.value = true;
  try {
    const result = await getIntegrationConfigList();
    configList.value = result.data?.configs || [];
  } catch (error) {
    console.error('Failed to fetch config list:', error);
    MessagePlugin.error('获取配置列表失败');
  } finally {
    loading.value = false;
  }
};

const showCreateDialog = () => {
  isEdit.value = false;
  Object.assign(formData, {
    platform: '',
    enabled: true,
    app_key: '',
    app_secret: '',
    agent_id: '',
    app_id: '',
    token: '',
    encoding_aes_key: '',
  });
  formVisible.value = true;
};

const handleEdit = (row: IntegrationConfig) => {
  isEdit.value = true;
  Object.assign(formData, {
    id: row.id,
    platform: row.platform,
    enabled: row.enabled,
    app_key: row.app_key || '',
    app_secret: '', // 不回显密码
    agent_id: row.agent_id || '',
    app_id: row.app_id || '',
    token: row.token || '',
    encoding_aes_key: row.encoding_aes_key || '',
  });
  formVisible.value = true;
};

const handleSubmit = async () => {
  const valid = await formRef.value?.validate();
  if (!valid) return;

  submitting.value = true;
  try {
    if (isEdit.value && formData.id) {
      await updateIntegrationConfig(formData.id, formData as IntegrationConfig);
      MessagePlugin.success('更新成功');
    } else {
      await createIntegrationConfig(formData as IntegrationConfig);
      MessagePlugin.success('创建成功');
    }
    formVisible.value = false;
    fetchConfigList();
  } catch (error) {
    console.error('Failed to save config:', error);
    MessagePlugin.error('保存失败');
  } finally {
    submitting.value = false;
  }
};

const handleDelete = async (id?: number) => {
  if (!id) return;

  try {
    await deleteIntegrationConfig(id);
    MessagePlugin.success('删除成功');
    fetchConfigList();
  } catch (error) {
    console.error('Failed to delete config:', error);
    MessagePlugin.error('删除失败');
  }
};

onMounted(() => {
  fetchConfigList();
});
</script>

<style scoped lang="less">
.integration-config {
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

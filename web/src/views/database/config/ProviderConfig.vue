<template>
  <div class="provider-config">
    <div class="section">
      <div class="section-header">
        <h3>云服务商配置</h3>
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

        <template #regions="{ row }">
          <t-tag v-for="(region, idx) in parseRegions(row.regions)" :key="idx" size="small" style="margin-right: 4px;">
            {{ region }}
          </t-tag>
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
      :header="isEdit ? '编辑云服务商配置' : '添加云服务商配置'"
      width="700px"
      :on-confirm="handleSubmit"
      :confirm-btn="{ content: '保存', theme: 'primary', loading: submitting }"
    >
      <t-form :data="formData" ref="formRef" :rules="formRules" label-width="120px">
        <t-form-item label="云服务商" name="provider">
          <t-select v-model="formData.provider" placeholder="请选择云服务商" :disabled="isEdit">
            <t-option value="aliyun" label="阿里云"></t-option>
            <t-option value="tencent" label="腾讯云"></t-option>
          </t-select>
        </t-form-item>

        <t-form-item label="账号名称" name="account_name">
          <t-input v-model="formData.account_name" placeholder="例如: default, prod" :disabled="isEdit" />
        </t-form-item>

        <t-form-item label="启用状态" name="enabled">
          <t-switch v-model="formData.enabled" />
        </t-form-item>

        <t-form-item label="AccessKey ID" name="access_key_id">
          <t-input v-model="formData.access_key_id" placeholder="请输入AccessKey ID" />
        </t-form-item>

        <t-form-item label="AccessKey Secret" name="access_key_secret">
          <t-input v-model="formData.access_key_secret" type="password" placeholder="请输入AccessKey Secret" />
        </t-form-item>

        <t-form-item label="区域列表" name="regions">
          <t-textarea
            v-model="formData.regions"
            placeholder='JSON数组格式,例如: ["cn-hangzhou","cn-shanghai"]'
            :autosize="{ minRows: 2, maxRows: 4 }"
          />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import {
  getProviderConfigList,
  createProviderConfig,
  updateProviderConfig,
  deleteProviderConfig,
  type ProviderConfig,
} from '@/api/config';
import { MessagePlugin } from 'tdesign-vue-next';

const loading = ref(false);
const submitting = ref(false);
const configList = ref<ProviderConfig[]>([]);
const formVisible = ref(false);
const isEdit = ref(false);
const formRef = ref();

const formData = reactive<Partial<ProviderConfig>>({
  provider: '',
  account_name: '',
  enabled: true,
  access_key_id: '',
  access_key_secret: '',
  regions: '',
});

const formRules = {
  provider: [{ required: true, message: '请选择云服务商' }],
  account_name: [{ required: true, message: '请输入账号名称' }],
  access_key_id: [{ required: true, message: '请输入AccessKey ID' }],
  access_key_secret: [{ required: true, message: '请输入AccessKey Secret' }],
  regions: [
    { required: true, message: '请输入区域列表' },
    {
      validator: (val: string) => {
        try {
          const parsed = JSON.parse(val);
          return Array.isArray(parsed);
        } catch {
          return false;
        }
      },
      message: '区域列表必须是有效的JSON数组格式',
    },
  ],
};

const columns = [
  { colKey: 'id', title: 'ID', width: 80 },
  { colKey: 'provider', title: '云服务商', width: 120 },
  { colKey: 'account_name', title: '账号名称', width: 150 },
  { colKey: 'enabled', title: '状态', width: 100, cell: 'enabled' },
  { colKey: 'access_key_id', title: 'AccessKey ID', ellipsis: true },
  { colKey: 'regions', title: '区域', cell: 'regions', width: 300 },
  { colKey: 'operation', title: '操作', width: 150, cell: 'operation' },
];

const fetchConfigList = async () => {
  loading.value = true;
  try {
    const result = await getProviderConfigList();
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
    provider: '',
    account_name: '',
    enabled: true,
    access_key_id: '',
    access_key_secret: '',
    regions: '',
  });
  formVisible.value = true;
};

const handleEdit = (row: ProviderConfig) => {
  isEdit.value = true;
  Object.assign(formData, {
    id: row.id,
    provider: row.provider,
    account_name: row.account_name,
    enabled: row.enabled,
    access_key_id: row.access_key_id,
    access_key_secret: '', // 不回显密码
    regions: row.regions,
  });
  formVisible.value = true;
};

const handleSubmit = async () => {
  const valid = await formRef.value?.validate();
  if (!valid) return;

  submitting.value = true;
  try {
    if (isEdit.value && formData.id) {
      await updateProviderConfig(formData.id, formData as ProviderConfig);
      MessagePlugin.success('更新成功');
    } else {
      await createProviderConfig(formData as ProviderConfig);
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
    await deleteProviderConfig(id);
    MessagePlugin.success('删除成功');
    fetchConfigList();
  } catch (error) {
    console.error('Failed to delete config:', error);
    MessagePlugin.error('删除失败');
  }
};

const parseRegions = (regionsStr: string): string[] => {
  try {
    return JSON.parse(regionsStr);
  } catch {
    return [];
  }
};

onMounted(() => {
  fetchConfigList();
});
</script>

<style scoped lang="less">
.provider-config {
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

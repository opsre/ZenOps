<template>
  <div class="mcp-page">
    <section class="hero-card">
      <div>
        <p class="hero-badge">MODEL CONTEXT PROTOCOL</p>
        <div class="hero-title">
          <h2>MCP Server 中枢面板</h2>
          <t-tag theme="primary" variant="outline">Beta</t-tag>
        </div>
        <p class="hero-desc">
          统一管理外部 MCP Server，快速完成增删改查、状态切换与工具调试，确保运维自动化链路稳定可靠。
        </p>
        <div class="hero-actions">
          <t-button theme="primary" size="medium" @click="handleAdd">
            <template #icon><t-icon name="add" /></template>
            新建 MCP Server
          </t-button>
          <t-button variant="outline" size="medium" @click="fetchServers" :loading="loading">
            <template #icon><t-icon name="refresh" /></template>
            立即同步
          </t-button>
        </div>
      </div>
      <div class="hero-insight">
        <div class="insight-item">
          <span>在线数量</span>
          <strong>{{ activeServers }}</strong>
          <p>共 {{ totalServers }} 个接入源</p>
        </div>
        <div class="divider"></div>
        <div class="insight-item">
          <span>长连接能力</span>
          <strong>{{ longRunningServers }}</strong>
          <p>{{ httpBasedServers }} 个 HTTP / SSE 接入</p>
        </div>
      </div>
    </section>

    <section class="stats-grid">
      <t-card class="stat-card" :bordered="false">
        <div class="stat-icon primary">
          <t-icon name="server" />
        </div>
        <div>
          <p class="stat-label">已接入 MCP</p>
          <p class="stat-value">{{ totalServers }}</p>
          <span class="stat-desc">覆盖 {{ availableTags.length }} 个标签分类</span>
        </div>
      </t-card>
      <t-card class="stat-card" :bordered="false">
        <div class="stat-icon success">
          <t-icon name="check-circle" />
        </div>
        <div>
          <p class="stat-label">运行中</p>
          <p class="stat-value">{{ activeServers }}</p>
          <span class="stat-desc">停用 {{ inactiveServers }} 个实例</span>
        </div>
      </t-card>
      <t-card class="stat-card" :bordered="false">
        <div class="stat-icon warning">
          <t-icon name="dashboard" />
        </div>
        <div>
          <p class="stat-label">Stdio Bridge</p>
          <p class="stat-value">{{ stdioServers }}</p>
          <span class="stat-desc">{{ httpBasedServers }} 个远程连接</span>
        </div>
      </t-card>
    </section>

    <t-card class="filter-card" :bordered="false">
      <div class="filter-grid">
        <t-input v-model.trim="searchKeyword" placeholder="搜索名称 / 描述" clearable>
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>

        <t-select v-model="statusFilter" placeholder="状态筛选">
          <t-option value="all" label="全部状态" />
          <t-option value="active" label="仅启用" />
          <t-option value="inactive" label="仅停用" />
        </t-select>

        <t-select v-model="typeFilter" placeholder="连接类型">
          <t-option value="all" label="全部类型" />
          <t-option value="stdio" label="Stdio" />
          <t-option value="sse" label="SSE" />
          <t-option value="streamableHttp" label="Streamable HTTP" />
        </t-select>

        <t-select v-model="tagFilter" placeholder="标签筛选" clearable @clear="tagFilter = 'all'">
          <t-option value="all" label="全部标签" />
          <t-option v-for="tag in availableTags" :key="tag" :value="tag" :label="tag" />
        </t-select>

        <t-button variant="text" class="reset-btn" @click="resetFilters">
          <template #icon><t-icon name="rollback" /></template>
          重置筛选
        </t-button>
      </div>
    </t-card>

    <t-card class="list-card" :bordered="false">
      <div class="list-header">
        <div>
          <h3>外部 MCP Server</h3>
          <p>实时查看状态、配置以及可用于代理的工具</p>
        </div>
        <div class="list-actions">
          <t-button theme="primary" size="medium" @click="handleAdd">
            <template #icon><t-icon name="add" /></template>
            新增
          </t-button>
          <t-button variant="outline" size="medium" @click="handleExport">
            <template #icon><t-icon name="download" /></template>
            导出配置
          </t-button>
          <t-upload
            theme="custom"
            accept=".json"
            :before-upload="handleImport"
            :auto-upload="false"
            style="display: inline-block"
          >
            <t-button variant="outline" size="medium">
              <template #icon><t-icon name="upload" /></template>
              导入配置
            </t-button>
          </t-upload>
          <t-button variant="outline" size="medium" @click="fetchServers" :loading="loading">
            <template #icon><t-icon name="refresh" /></template>
            刷新
          </t-button>
        </div>
      </div>

      <t-table
        class="mcp-table"
        row-key="name"
        :data="filteredServers"
        :columns="columns"
        :loading="loading"
        stripe
        hover
      >
        <template #name="{ row }">
          <div class="name-cell">
            <div class="name-line">
              <span class="name-text">{{ row.name }}</span>
              <t-tag v-if="row.toolPrefix" size="small" theme="primary" variant="light">
                {{ row.toolPrefix }}
              </t-tag>
              <t-tag v-if="row.longRunning" size="small" theme="success" variant="outline">
                常驻
              </t-tag>
            </div>
            <p class="desc-text">{{ row.description || '暂无描述' }}</p>
          </div>
        </template>

        <template #provider="{ row }">
          <div class="provider-cell">
            <span>{{ row.provider }}</span>
            <t-link v-if="row.providerUrl" :href="row.providerUrl" target="_blank" theme="primary">官网</t-link>
          </div>
        </template>

        <template #type="{ row }">
          <t-tag :theme="row.type === 'stdio' ? 'default' : 'primary'" variant="light-outline">
            {{ row.type }}
          </t-tag>
        </template>

        <template #timeout="{ row }">
          <div class="timeout-cell">
            <span>{{ row.timeout }} s</span>
            <t-progress :percentage="Math.min(100, (row.timeout / 600) * 100)" size="small" theme="plump" />
          </div>
        </template>

        <template #tags="{ row }">
          <t-tag
            v-for="tag in row.tags"
            :key="tag"
            theme="primary"
            variant="light"
            size="small"
            style="margin-right: 4px"
          >
            {{ tag }}
          </t-tag>
        </template>

        <template #isActive="{ row }">
          <t-switch :value="row.isActive" :before-change="(val: boolean) => handleToggleSwitch(row, val)" />
        </template>

        <template #op="{ row }">
          <t-space>
            <t-link theme="primary" @click="handleEdit(row)">编辑</t-link>
            <t-link theme="primary" @click="handleViewTools(row)">工具</t-link>
            <t-popconfirm content="确认删除该 MCP Server 吗？" @confirm="handleDelete(row)">
              <t-link theme="danger">删除</t-link>
            </t-popconfirm>
          </t-space>
        </template>

        <template #empty>
          <div class="empty-state">
            <t-icon name="server" size="48" />
            <p>还没有任何外部 MCP Server，点击下方按钮快速接入。</p>
            <t-button theme="primary" @click="handleAdd">立即添加</t-button>
          </div>
        </template>
      </t-table>
    </t-card>

    <t-dialog
      v-model:visible="dialogVisible"
      :header="isEdit ? '编辑 MCP Server' : '添加 MCP Server'"
      width="640px"
      @confirm="handleDialogConfirm"
    >
      <t-form ref="formRef" :data="formData" :rules="rules" label-align="top">
        <t-form-item label="名称" name="name">
          <t-input v-model="formData.name" :disabled="isEdit" placeholder="请输入 MCP Server 名称" />
        </t-form-item>
        <t-form-item label="类型" name="type">
          <t-select v-model="formData.type" placeholder="请选择连接类型">
            <t-option label="Stdio (本地命令)" value="stdio" />
            <t-option label="SSE (Server-Sent Events)" value="sse" />
            <t-option label="Streamable HTTP" value="streamableHttp" />
          </t-select>
        </t-form-item>
        <t-form-item label="描述" name="description">
          <t-textarea v-model="formData.description" placeholder="请输入描述" />
        </t-form-item>

        <template v-if="formData.type === 'stdio'">
          <t-form-item label="命令" name="command">
            <t-input v-model="formData.command" placeholder="例如: python3, npx" />
          </t-form-item>
          <t-form-item label="参数" name="args">
            <t-tag-input v-model="formData.args" placeholder="输入参数后按回车" />
          </t-form-item>
          <t-form-item label="环境变量" name="env">
            <t-textarea
              :value="envString"
              @change="handleEnvChange"
              placeholder="KEY=VALUE (每行一个)"
              :autosize="{ minRows: 3, maxRows: 5 }"
            />
          </t-form-item>
        </template>

        <template v-if="formData.type && ['sse', 'streamableHttp'].includes(formData.type)">
          <t-form-item label="Base URL" name="baseUrl">
            <t-input v-model="formData.baseUrl" placeholder="http://localhost:8080" />
          </t-form-item>
        </template>

        <t-divider />

        <t-form-item label="提供商信息">
          <t-input-group>
            <t-input v-model="formData.provider" placeholder="提供商名称" style="width: 50%" />
            <t-input v-model="formData.providerUrl" placeholder="提供商 URL" style="width: 50%" />
          </t-input-group>
        </t-form-item>

        <t-form-item label="超时时间(秒)" name="timeout">
          <t-input-number v-model="formData.timeout" :min="1" :max="3600" placeholder="300" style="width: 100%" />
        </t-form-item>

        <t-form-item label="工具前缀" name="toolPrefix">
          <t-input
            v-model="formData.toolPrefix"
            placeholder="留空自动生成"
            :tips="`工具将以 '${formData.toolPrefix || formData.name + '_'}' 为前缀注册到系统`"
          />
        </t-form-item>

        <t-form-item label="标签" name="tags">
          <t-tag-input v-model="formData.tags" placeholder="输入标签后按回车，如: cicd, git" />
        </t-form-item>

        <t-form-item label="高级选项">
          <t-space direction="vertical" style="width: 100%">
            <t-checkbox v-model="formData.longRunning">长期运行进程</t-checkbox>
            <t-checkbox v-model="formData.autoRegister">自动注册工具到 ZenOps MCP</t-checkbox>
            <t-checkbox v-model="formData.isActive">创建后立即启用</t-checkbox>
          </t-space>
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-drawer v-model:visible="drawerVisible" header="工具列表" size="large" :footer="false">
      <div v-if="currentServer">
        <div class="drawer-header">
          <div>
            <h3>{{ currentServer.name }}</h3>
            <p class="desc-text">{{ currentServer.description }}</p>
          </div>
          <t-tag theme="success" variant="outline">{{ currentServer.type }}</t-tag>
        </div>

        <t-loading :loading="toolsLoading">
          <t-list :split="true">
            <t-list-item v-for="tool in tools" :key="tool.name">
              <t-list-item-meta :title="tool.name" :description="tool.description" />
              <template #action>
                <t-button variant="text" theme="primary" @click="openTestDialog(tool)">调试</t-button>
              </template>
            </t-list-item>
            <div v-if="tools.length === 0" class="empty-tools">暂无工具或获取失败</div>
          </t-list>
        </t-loading>
      </div>
    </t-drawer>

    <t-dialog
      v-model:visible="testDialogVisible"
      :header="`调试工具: ${currentTool?.name}`"
      width="720px"
      @confirm="handleTestTool"
      :confirm-btn="{ content: '执行', loading: testLoading, disabled: !!jsonError }"
    >
      <div v-if="currentTool">
        <t-alert v-if="currentTool.description" theme="info" :message="currentTool.description" style="margin-bottom: 16px" />

        <div class="schema-viewer">
          <h4>参数定义</h4>
          <t-collapse default-value="[]">
            <t-collapse-panel value="schema" header="点击展开查看参数 Schema">
              <pre>{{ JSON.stringify(currentTool.inputSchema, null, 2) }}</pre>
            </t-collapse-panel>
          </t-collapse>
        </div>

        <t-form label-align="top" style="margin-top: 16px">
          <t-form-item label="输入参数 (JSON)" :status="jsonError ? 'error' : 'default'" :tips="jsonError">
            <t-textarea
              v-model="testArgs"
              placeholder='{ "arg": "value" }'
              :autosize="{ minRows: 6, maxRows: 12 }"
              @blur="validateJSON"
            />
          </t-form-item>
        </t-form>

        <div v-if="testResult" class="test-result">
          <t-divider />
          <h4>执行结果</h4>
          <t-alert
            v-if="testResult.isError"
            theme="error"
            message="工具执行失败"
            style="margin-bottom: 12px"
          />
          <t-tabs default-value="formatted">
            <t-tab-panel value="formatted" label="格式化显示">
              <div v-for="(content, index) in testResult.content" :key="index" class="result-item">
                <t-tag v-if="content.type" size="small" style="margin-bottom: 8px">{{ content.type }}</t-tag>
                <pre class="result-text">{{ content.text }}</pre>
              </div>
            </t-tab-panel>
            <t-tab-panel value="raw" label="原始数据">
              <pre>{{ JSON.stringify(testResult, null, 2) }}</pre>
            </t-tab-panel>
          </t-tabs>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useMCPStore } from '@/store/mcp';
import { Icon as TIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import type { MCPServerConfig, MCPTool } from '@/types/mcp';

const store = useMCPStore();
const { servers, loading, tools } = storeToRefs(store);

const toolsLoading = ref(false);
const dialogVisible = ref(false);
const isEdit = ref(false);
const formRef = ref();
const formData = ref<Partial<MCPServerConfig>>({
  name: '',
  type: 'stdio',
  description: '',
  command: '',
  args: [],
  env: {},
  baseUrl: '',
  tags: [],
  isActive: false,
  timeout: 300,
  toolPrefix: '',
  autoRegister: true,
});
const envString = ref('');

const searchKeyword = ref('');
const statusFilter = ref<'all' | 'active' | 'inactive'>('all');
const typeFilter = ref<'all' | MCPServerConfig['type']>('all');
const tagFilter = ref<string>('all');

const availableTags = computed(() => {
  const tagSet = new Set<string>();
  servers.value.forEach((server) => server.tags?.forEach((tag) => tagSet.add(tag)));
  return Array.from(tagSet);
});

const filteredServers = computed<MCPServerConfig[]>(() => {
  return servers.value.filter((server) => {
    const keyword = searchKeyword.value.toLowerCase();
    const keywordMatch =
      !keyword ||
      server.name.toLowerCase().includes(keyword) ||
      (server.description || '').toLowerCase().includes(keyword);

    const statusMatch =
      statusFilter.value === 'all' ||
      (statusFilter.value === 'active' ? server.isActive : !server.isActive);

    const typeMatch = typeFilter.value === 'all' || server.type === typeFilter.value;

    const tagValue = tagFilter.value;
    const tagMatch = !tagValue || tagValue === 'all' || server.tags?.includes(tagValue);

    return keywordMatch && statusMatch && typeMatch && tagMatch;
  });
});

const totalServers = computed(() => servers.value.length);
const activeServers = computed(() => servers.value.filter((server) => server.isActive).length);
const inactiveServers = computed(() => totalServers.value - activeServers.value);
const stdioServers = computed(() => servers.value.filter((server) => server.type === 'stdio').length);
const httpBasedServers = computed(() => servers.value.filter((server) => server.type !== 'stdio').length);
const longRunningServers = computed(() => servers.value.filter((server) => server.longRunning).length);

const columns = [
  { colKey: 'name', title: '名称 / 描述', width: 260 },
  { colKey: 'provider', title: '提供商', width: 180 },
  { colKey: 'type', title: '连接方式', width: 140 },
  { colKey: 'timeout', title: '超时时间', width: 160 },
  { colKey: 'tags', title: '标签', width: 220 },
  { colKey: 'isActive', title: '状态', width: 120 },
  { colKey: 'op', title: '操作', width: 200, fixed: 'right' },
];

const rules = {
  name: [
    { required: true, message: '请输入名称', trigger: 'blur' },
    {
      validator: (val: string) => {
        if (!/^[a-zA-Z0-9_-]+$/.test(val)) {
          return { result: false, message: '名称只能包含字母、数字、下划线和连字符', type: 'error' };
        }
        return { result: true };
      },
      trigger: 'blur',
    },
  ],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  description: [
    {
      validator: (val: string) => {
        if (val && val.length > 200) {
          return { result: false, message: '描述不能超过200个字符', type: 'error' };
        }
        return { result: true };
      },
      trigger: 'blur',
    },
  ],
  command: [
    {
      validator: (val: string) => {
        if (formData.value.type === 'stdio' && !val) {
          return { result: false, message: 'Stdio 类型必须输入命令', type: 'error' };
        }
        return { result: true };
      },
      trigger: 'blur',
    },
  ],
  baseUrl: [
    {
      validator: (val: string) => {
        if (['sse', 'streamableHttp'].includes(formData.value.type || '') && !val) {
          return { result: false, message: 'SSE/HTTP 类型必须输入 Base URL', type: 'error' };
        }
        if (val && !/^https?:\/\/.+/.test(val)) {
          return { result: false, message: 'URL 格式不正确，需以 http:// 或 https:// 开头', type: 'error' };
        }
        return { result: true };
      },
      trigger: 'blur',
    },
  ],
  timeout: [
    {
      validator: (val: number) => {
        if (val && (val < 1 || val > 3600)) {
          return { result: false, message: '超时时间应在 1-3600 秒之间', type: 'error' };
        }
        return { result: true };
      },
      trigger: 'blur',
    },
  ],
};

const fetchServers = () => store.fetchServers();

const resetFilters = () => {
  searchKeyword.value = '';
  statusFilter.value = 'all';
  typeFilter.value = 'all';
  tagFilter.value = 'all';
};

// 使用 before-change 钩子处理 Switch 切换
const handleToggleSwitch = async (row: MCPServerConfig, val: boolean): Promise<boolean> => {
  const success = await store.toggleServer(row.name, val);
  return success; // 返回 true 才会切换状态
};

const handleDelete = async (row: MCPServerConfig) => {
  await store.deleteServer(row.name);
};

const handleAdd = () => {
  isEdit.value = false;
  formData.value = {
    name: '',
    type: 'stdio',
    description: '',
    command: '',
    args: [],
    env: {},
    baseUrl: '',
    tags: [],
    isActive: false,
    timeout: 300,
    toolPrefix: '',
    autoRegister: true,
    provider: '',
    providerUrl: '',
    longRunning: false,
    logoUrl: '',
    installSource: 'manual',
  };
  envString.value = '';
  dialogVisible.value = true;
};

const handleEdit = (row: MCPServerConfig) => {
  isEdit.value = true;
  formData.value = JSON.parse(JSON.stringify(row));
  envString.value = row.env
    ? Object.entries(row.env)
        .map(([k, v]) => `${k}=${v}`)
        .join('\n')
    : '';
  dialogVisible.value = true;
};

const handleEnvChange = (val: string) => {
  envString.value = val;
  const env: Record<string, string> = {};
  val.split('\n').forEach((line) => {
    const [k, ...v] = line.split('=');
    if (k && v.length) {
      env[k.trim()] = v.join('=').trim();
    }
  });
  formData.value.env = env;
};

const handleDialogConfirm = async () => {
  const valid = await formRef.value.validate();
  if (valid !== true) return;

  const config = formData.value as MCPServerConfig;
  let success = false;
  if (isEdit.value) {
    success = await store.updateServer(config.name, config);
  } else {
    success = await store.addServer(config);
  }

  if (success) {
    dialogVisible.value = false;
  }
};

const drawerVisible = ref(false);
const currentServer = ref<MCPServerConfig | null>(null);

const handleViewTools = async (row: MCPServerConfig) => {
  currentServer.value = row;
  drawerVisible.value = true;
  toolsLoading.value = true;
  await store.fetchTools(row.name);
  toolsLoading.value = false;
};

const testDialogVisible = ref(false);
const currentTool = ref<MCPTool | null>(null);
const testArgs = ref('{}');
const testResult = ref<any>(null);
const testLoading = ref(false);
const jsonError = ref('');

const openTestDialog = (tool: MCPTool) => {
  currentTool.value = tool;
  testArgs.value = '{}';
  testResult.value = null;
  jsonError.value = '';
  testDialogVisible.value = true;
};

// 验证 JSON 格式
const validateJSON = () => {
  try {
    JSON.parse(testArgs.value);
    jsonError.value = '';
  } catch (error) {
    jsonError.value = 'JSON 格式错误，请检查语法';
  }
};

const handleTestTool = async () => {
  if (!currentServer.value || !currentTool.value) return;

  // 再次验证 JSON
  let args = {};
  try {
    args = JSON.parse(testArgs.value);
  } catch (error) {
    MessagePlugin.error('参数格式错误，请输入有效的 JSON');
    return;
  }

  testLoading.value = true;
  try {
    const result = await store.testTool(currentServer.value.name, currentTool.value.name, args);
    testResult.value = result;
    MessagePlugin.success('工具执行成功');
  } catch (error) {
    // 错误已在 store 中处理
  } finally {
    testLoading.value = false;
  }
};

// 导出配置
const handleExport = () => {
  if (servers.value.length === 0) {
    MessagePlugin.warning('暂无配置可导出');
    return;
  }

  const config = {
    mcpServers: servers.value.reduce((acc, server) => {
      acc[server.name] = server;
      return acc;
    }, {} as Record<string, MCPServerConfig>),
  };

  const dataStr = JSON.stringify(config, null, 2);
  const blob = new Blob([dataStr], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `mcp_servers_${new Date().getTime()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  MessagePlugin.success('配置已导出');
};

// 导入配置
const handleImport = async (file: File) => {
  try {
    const text = await file.text();
    const config = JSON.parse(text);

    if (!config.mcpServers || typeof config.mcpServers !== 'object') {
      MessagePlugin.error('配置文件格式错误，缺少 mcpServers 字段');
      return false;
    }

    const serversArray = Object.values(config.mcpServers) as MCPServerConfig[];

    if (serversArray.length === 0) {
      MessagePlugin.warning('配置文件中没有可导入的 MCP Server');
      return false;
    }

    let successCount = 0;
    let failCount = 0;

    for (const server of serversArray) {
      try {
        await store.addServer(server);
        successCount++;
      } catch (error) {
        failCount++;
        console.error(`Failed to import ${server.name}:`, error);
      }
    }

    if (successCount > 0) {
      MessagePlugin.success(`成功导入 ${successCount} 个配置${failCount > 0 ? `，失败 ${failCount} 个` : ''}`);
      await fetchServers();
    } else {
      MessagePlugin.error('导入失败，请检查配置文件');
    }
  } catch (error) {
    MessagePlugin.error('配置文件解析失败: ' + (error as Error).message);
  }

  return false; // 阻止自动上传
};

onMounted(() => {
  fetchServers();
});
</script>

<style scoped>
.mcp-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: #0f172a;
}

.hero-card {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  padding: 20px 24px;
  border-radius: 16px;
  background: linear-gradient(135deg, #f0f6ff, #fff);
  border: 1px solid rgba(37, 99, 235, 0.12);
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.04);
}

.hero-badge {
  font-size: 11px;
  letter-spacing: 0.15em;
  color: #64748b;
  margin-bottom: 8px;
  font-weight: 500;
}

.hero-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.hero-title h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: #0f172a;
}

.hero-desc {
  margin: 8px 0 16px;
  color: #64748b;
  line-height: 1.5;
  font-size: 14px;
}

.hero-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.hero-insight {
  min-width: 220px;
  display: flex;
  align-items: center;
  gap: 20px;
  padding-left: 20px;
  border-left: 1px solid rgba(37, 99, 235, 0.12);
}

.insight-item span {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}

.insight-item strong {
  font-size: 26px;
  line-height: 1;
  color: #0f172a;
  font-weight: 700;
}

.insight-item p {
  margin: 0;
  color: #94a3b8;
  font-size: 12px;
}

.divider {
  width: 1px;
  align-self: stretch;
  background: rgba(37, 99, 235, 0.12);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border-radius: 12px;
  background: #fff;
  border: 1px solid rgba(148, 163, 184, 0.15);
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
  transition: all 0.2s ease;
}

.stat-card:hover {
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08);
  border-color: rgba(148, 163, 184, 0.25);
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  color: #fff;
  font-size: 20px;
  flex-shrink: 0;
}

.stat-icon.primary {
  background: linear-gradient(135deg, #2563eb, #60a5fa);
}

.stat-icon.success {
  background: linear-gradient(135deg, #10b981, #34d399);
}

.stat-icon.warning {
  background: linear-gradient(135deg, #f59e0b, #fcd34d);
}

.stat-label {
  margin: 0;
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}

.stat-value {
  margin: 4px 0 2px;
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1;
}

.stat-desc {
  font-size: 11px;
  color: #94a3b8;
}

.filter-card {
  border-radius: 12px;
  background: #fff;
  border: 1px solid rgba(148, 163, 184, 0.15);
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
  padding: 16px;
}

.filter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.reset-btn {
  justify-self: flex-end;
  color: #6366f1;
}

.list-card {
  border-radius: 12px;
  background: #fff;
  border: 1px solid rgba(148, 163, 184, 0.15);
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
  padding: 20px;
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.list-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #0f172a;
}

.list-header p {
  margin: 4px 0 0;
  font-size: 13px;
  color: #94a3b8;
}

.list-actions {
  display: flex;
  gap: 12px;
}

.mcp-table :deep(.t-table__content) {
  background: transparent;
}

.name-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.name-text {
  font-weight: 600;
  color: #111827;
}

.desc-text {
  margin: 0;
  font-size: 13px;
  color: #94a3b8;
}

.provider-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.timeout-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.empty-state {
  padding: 48px 0;
  text-align: center;
  color: #94a3b8;
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.empty-tools {
  text-align: center;
  color: #94a3b8;
  padding: 32px 0;
}

.schema-viewer {
  margin-bottom: 16px;
}

.schema-viewer h4,
.test-result h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
}

.schema-viewer pre {
  margin: 0;
  padding: 12px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  white-space: pre-wrap;
  font-size: 12px;
  color: #0f172a;
  overflow-x: auto;
}

.test-result {
  margin-top: 24px;
}

.result-item {
  margin-bottom: 12px;
}

.result-text {
  margin: 0;
  padding: 12px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  white-space: pre-wrap;
  font-size: 12px;
  color: #0f172a;
  line-height: 1.6;
}

.tool-desc {
  margin: 0 0 16px 0;
  color: #475569;
  line-height: 1.6;
}

@media (max-width: 960px) {
  .hero-card {
    flex-direction: column;
  }

  .hero-insight {
    border-left: none;
    border-top: 1px solid rgba(37, 99, 235, 0.15);
    padding-left: 0;
    padding-top: 24px;
    width: 100%;
    justify-content: space-between;
  }

  .list-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .list-actions {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>

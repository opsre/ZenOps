<template>
  <div class="request-logs-page">
    <t-card title="请求日志" :bordered="false">
      <!-- 搜索区域 -->
      <t-form :data="searchForm" layout="inline" class="search-form">
        <t-form-item label="来源">
          <t-select v-model="searchForm.source" clearable placeholder="请选择来源" style="width: 150px">
            <t-option value="http" label="HTTP"></t-option>
            <t-option value="mcp" label="MCP"></t-option>
            <t-option value="dingtalk" label="钉钉"></t-option>
            <t-option value="feishu" label="飞书"></t-option>
            <t-option value="wecom" label="企业微信"></t-option>
          </t-select>
        </t-form-item>

        <t-form-item label="状态码">
          <t-input-number v-model="searchForm.status_code" clearable placeholder="状态码" style="width: 120px" />
        </t-form-item>

        <t-form-item label="时间范围">
          <t-date-range-picker
            v-model="searchForm.dateRange"
            clearable
            allow-input
            format="YYYY-MM-DD"
            placeholder="选择日期范围"
            style="width: 300px"
          />
        </t-form-item>

        <t-form-item>
          <t-button theme="primary" @click="fetchLogs">查询</t-button>
          <t-button theme="default" @click="resetSearch">重置</t-button>
        </t-form-item>
      </t-form>

      <!-- 统计卡片 -->
      <div class="stats-cards" v-if="stats">
        <t-row :gutter="16">
          <t-col :span="3">
            <t-card :bordered="false" class="stat-card">
              <div class="stat-title">总请求数</div>
              <div class="stat-value">{{ stats.total_count }}</div>
            </t-card>
          </t-col>
          <t-col :span="3">
            <t-card :bordered="false" class="stat-card">
              <div class="stat-title">平均耗时</div>
              <div class="stat-value">{{ stats.avg_duration.toFixed(2) }}ms</div>
            </t-card>
          </t-col>
          <t-col :span="6">
            <t-card :bordered="false" class="stat-card">
              <div class="stat-title">按来源统计</div>
              <div class="stat-list">
                <div v-for="item in stats.source_stats" :key="item.source" class="stat-item">
                  {{ item.source }}: {{ item.count }}
                </div>
              </div>
            </t-card>
          </t-col>
        </t-row>
      </div>

      <!-- 表格 -->
      <t-table
        :data="logs"
        :columns="columns"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        stripe
        @page-change="onPageChange"
      >
        <template #source="{ row }">
          <t-tag :theme="getSourceTagTheme(row.source)">{{ row.source }}</t-tag>
        </template>

        <template #status_code="{ row }">
          <t-tag :theme="getStatusTagTheme(row.status_code)">{{ row.status_code }}</t-tag>
        </template>

        <template #duration="{ row }">
          {{ row.duration }}ms
        </template>

        <template #created_at="{ row }">
          {{ formatDate(row.created_at) }}
        </template>

        <template #operation="{ row }">
          <t-button theme="primary" variant="text" size="small" @click="viewDetail(row)">查看详情</t-button>
        </template>
      </t-table>
    </t-card>

    <!-- 详情对话框 -->
    <t-dialog
      v-model:visible="detailVisible"
      header="请求日志详情"
      width="800px"
      :footer="false"
    >
      <div v-if="currentLog" class="log-detail">
        <t-descriptions :data="detailData" :column="2" />

        <t-divider />

        <div v-if="currentLog.request_body">
          <h4>请求体</h4>
          <pre class="json-code">{{ formatJSON(currentLog.request_body) }}</pre>
        </div>

        <div v-if="currentLog.response_body">
          <h4>响应体</h4>
          <pre class="json-code">{{ formatJSON(currentLog.response_body) }}</pre>
        </div>

        <div v-if="currentLog.error">
          <h4>错误信息</h4>
          <t-alert theme="error" :message="currentLog.error" />
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { getRequestLogs, getRequestLogStats, type RequestLog, type RequestLogStats } from '@/api/database';
import { MessagePlugin } from 'tdesign-vue-next';

const loading = ref(false);
const logs = ref<RequestLog[]>([]);
const stats = ref<RequestLogStats | null>(null);
const detailVisible = ref(false);
const currentLog = ref<RequestLog | null>(null);

const searchForm = reactive({
  source: '',
  status_code: undefined as number | undefined,
  dateRange: [] as string[],
});

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
});

const columns = [
  { colKey: 'id', title: 'ID', width: 80 },
  { colKey: 'source', title: '来源', width: 100, cell: 'source' },
  { colKey: 'method', title: '方法', width: 100 },
  { colKey: 'path', title: '路径', ellipsis: true },
  { colKey: 'user_name', title: '用户', width: 120 },
  { colKey: 'status_code', title: '状态码', width: 100, cell: 'status_code' },
  { colKey: 'duration', title: '耗时', width: 100, cell: 'duration' },
  { colKey: 'created_at', title: '创建时间', width: 180, cell: 'created_at' },
  { colKey: 'operation', title: '操作', width: 100, cell: 'operation' },
];

const detailData = computed(() => {
  if (!currentLog.value) return [];

  return [
    { label: '请求ID', value: currentLog.value.request_id },
    { label: '来源', value: currentLog.value.source },
    { label: '方法', value: currentLog.value.method },
    { label: '路径', value: currentLog.value.path },
    { label: '用户ID', value: currentLog.value.user_id || '-' },
    { label: '用户名', value: currentLog.value.user_name || '-' },
    { label: '状态码', value: currentLog.value.status_code },
    { label: '耗时', value: `${currentLog.value.duration}ms` },
    { label: 'IP地址', value: currentLog.value.ip || '-' },
    { label: 'User-Agent', value: currentLog.value.user_agent || '-' },
    { label: '创建时间', value: formatDate(currentLog.value.created_at) },
  ];
});

const fetchLogs = async () => {
  loading.value = true;
  try {
    const params: any = {
      page: pagination.current,
      page_size: pagination.pageSize,
    };

    if (searchForm.source) params.source = searchForm.source;
    if (searchForm.status_code) params.status_code = searchForm.status_code;
    if (searchForm.dateRange && searchForm.dateRange.length === 2) {
      params.start_time = searchForm.dateRange[0];
      params.end_time = searchForm.dateRange[1];
    }

    const result = await getRequestLogs(params);
    logs.value = result.data;
    pagination.total = result.total;
  } catch (error) {
    console.error('Failed to fetch logs:', error);
  } finally {
    loading.value = false;
  }
};

const fetchStats = async () => {
  try {
    const params: any = {};
    if (searchForm.dateRange && searchForm.dateRange.length === 2) {
      params.start_time = searchForm.dateRange[0];
      params.end_time = searchForm.dateRange[1];
    }
    stats.value = await getRequestLogStats(params);
  } catch (error) {
    console.error('Failed to fetch stats:', error);
  }
};

const resetSearch = () => {
  searchForm.source = '';
  searchForm.status_code = undefined;
  searchForm.dateRange = [];
  pagination.current = 1;
  fetchLogs();
  fetchStats();
};

const onPageChange = (pageInfo: any) => {
  pagination.current = pageInfo.current;
  pagination.pageSize = pageInfo.pageSize;
  fetchLogs();
};

const viewDetail = (log: RequestLog) => {
  currentLog.value = log;
  detailVisible.value = true;
};

const getSourceTagTheme = (source: string) => {
  const themes: Record<string, string> = {
    http: 'primary',
    mcp: 'success',
    dingtalk: 'warning',
    feishu: 'danger',
    wecom: 'default',
  };
  return themes[source] || 'default';
};

const getStatusTagTheme = (statusCode: number) => {
  if (statusCode >= 200 && statusCode < 300) return 'success';
  if (statusCode >= 300 && statusCode < 400) return 'warning';
  if (statusCode >= 400) return 'danger';
  return 'default';
};

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString('zh-CN');
};

const formatJSON = (jsonStr: string) => {
  try {
    const obj = JSON.parse(jsonStr);
    return JSON.stringify(obj, null, 2);
  } catch {
    return jsonStr;
  }
};

onMounted(() => {
  fetchLogs();
  fetchStats();
});
</script>

<style scoped lang="less">
.request-logs-page {
  padding: 24px;
}

.search-form {
  margin-bottom: 24px;
}

.stats-cards {
  margin-bottom: 24px;

  .stat-card {
    text-align: center;

    .stat-title {
      font-size: 14px;
      color: #666;
      margin-bottom: 8px;
    }

    .stat-value {
      font-size: 24px;
      font-weight: bold;
      color: #333;
    }

    .stat-list {
      font-size: 12px;

      .stat-item {
        margin: 4px 0;
      }
    }
  }
}

.log-detail {
  :deep(.t-descriptions__item-label) {
    font-weight: bold;
  }

  h4 {
    margin: 16px 0 8px;
  }

  .json-code {
    background: #f5f5f5;
    padding: 12px;
    border-radius: 4px;
    overflow-x: auto;
    font-size: 12px;
  }
}
</style>

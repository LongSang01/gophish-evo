<template>
  <div class="campaign-detail-container">
    <a-page-header :title="campaign.name" @back="router.back()">
      <template #tags>
        <a-tag :color="getStatusColor(campaign.status)">
          {{ getStatusText(campaign.status) }}
        </a-tag>
      </template>
      <template #extra>
        <a-space>
          <a-button @click="handleDuplicate">
            <CopyOutlined /> 复制配置
          </a-button>
          <a-button 
            v-if="campaign.status === 'In progress' || campaign.status === 'Queued'" 
            type="primary"
            danger 
            @click="handleComplete"
          >
            <StopOutlined /> 标记完成
          </a-button>
          <a-button 
            v-if="campaign.status === 'Scheduled'" 
            type="primary" 
            @click="handleLaunch"
          >
            <PlayCircleOutlined /> 立即启动
          </a-button>
        </a-space>
      </template>
    </a-page-header>

    <a-row :gutter="16" style="padding: 0 24px">
      <a-col :span="16">
        <a-card title="活动统计">
          <a-row :gutter="16">
            <a-col :span="8">
              <a-statistic title="邮件已发送" :value="campaign.stats?.sent || 0" />
            </a-col>
            <a-col :span="8">
              <a-statistic title="邮件已打开" :value="campaign.stats?.opened || 0" />
            </a-col>
            <a-col :span="8">
              <a-statistic title="链接已点击" :value="campaign.stats?.clicked || 0" />
            </a-col>
          </a-row>
          <div ref="resultChart" style="height: 300px; margin-top: 24px"></div>
        </a-card>
      </a-col>
      
      <a-col :span="8">
        <a-card title="活动详情">
          <a-descriptions :column="1" size="small">
            <a-descriptions-item label="创建时间">
              {{ formatDate(campaign.created_date) }}
            </a-descriptions-item>
            <a-descriptions-item label="启动时间">
              {{ formatDate(campaign.launch_date) }}
            </a-descriptions-item>
            <a-descriptions-item label="发送配置">
              {{ campaign.smtps?.[0]?.name }}
            </a-descriptions-item>
            <a-descriptions-item label="邮件模板">
              {{ campaign.template?.name }}
            </a-descriptions-item>
          <a-descriptions-item label="用户组">
            <a-tag v-for="g in campaign.groups" :key="g.id">{{ g.name }}</a-tag>
            <span v-if="!campaign.groups?.length">无</span>
          </a-descriptions-item>
          <a-descriptions-item label="发送数">
            <a-statistic :value="campaign.stats?.sent || 0" :value-style="{ fontSize: '16px' }" />
          </a-descriptions-item>
          <a-descriptions-item label="打开数">
            <a-statistic :value="campaign.stats?.opened || 0" :value-style="{ fontSize: '16px' }" />
          </a-descriptions-item>
          <a-descriptions-item label="点击数">
            <a-statistic :value="campaign.stats?.clicked || 0" :value-style="{ fontSize: '16px' }" />
          </a-descriptions-item>
          <a-descriptions-item label="提交数">
            <a-statistic :value="campaign.stats?.submitted_data || 0" :value-style="{ fontSize: '16px' }" />
          </a-descriptions-item>
          <a-descriptions-item label="报告数">
            <a-statistic :value="campaign.stats?.email_reported || 0" :value-style="{ fontSize: '16px' }" />
            </a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="用户结果" style="margin: 16px 24px">
      <template #extra>
        <a-space>
          <a-button size="small" @click="exportCSV('results')">导出结果CSV</a-button>
          <a-button size="small" @click="exportCSV('events')">导出事件CSV</a-button>
        </a-space>
      </template>
      <a-table
        :columns="resultColumns"
        :data-source="results"
        :loading="loadingResults"
        :pagination="{ pageSize: 20 }"
        row-key="id"
        :custom-row="(record: any) => ({ onClick: () => showRecipientTimeline(record) })"
        :row-class-name="() => 'clickable-row'"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'email'">
            <a>{{ record.email }}</a>
          </template>
          <template v-if="column.key === 'status'">
            <a-tag :color="getResultStatusColor(record.status)">
              {{ getResultStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'reported'">
            <a-tag :color="record.reported ? 'green' : 'default'">
              {{ record.reported ? '是' : '否' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'send_date'">
            {{ formatDate(record.send_date) }}
          </template>
          <template v-if="column.key === 'smtp_from_address'">
            {{ record.smtp_from_address || '-' }}
          </template>
          <template v-if="column.key === 'modified_date'">
            {{ formatDate(record.modified_date) }}
          </template>
        </template>
      </a-table>
    </a-card>

    <a-drawer
      v-model:open="drawerVisible"
      :title="`事件时间线 - ${drawerEmail}`"
      width="500"
    >
      <a-timeline>
        <a-timeline-item
          v-for="(event, index) in drawerEvents"
          :key="index"
          :color="getTimelineColor(event.message)"
        >
          <div class="event-card">
            <div class="event-header">
              <span class="event-type" :class="getEventClass(event.message)">{{ getEventText(event.message) }}</span>
              <span class="event-time">{{ formatDate(event.time) }}</span>
            </div>
            <div v-if="event.details" class="timeline-details">
              <template v-if="parseDetails(event.details)">
                <div v-for="(item, idx) in parseDetails(event.details)" :key="idx" class="detail-item">
                  <div class="detail-key">{{ item.key }}</div>
                  <div class="detail-value" :class="{ 'detail-truncate': item.value.length > 80 }" :title="item.value">{{ item.value }}</div>
                </div>
              </template>
              <pre v-else class="detail-raw">{{ event.details }}</pre>
            </div>
          </div>
        </a-timeline-item>
        <a-empty v-if="!drawerEvents.length" description="暂无事件" />
      </a-timeline>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { message, Modal } from 'ant-design-vue';
import { CopyOutlined, StopOutlined, PlayCircleOutlined } from '@ant-design/icons-vue';
import * as echarts from 'echarts';
import { getCampaign, getCampaignResults, completeCampaign, launchCampaign } from '@/api/campaigns';
import { formatDate } from '@/utils/format';

const route = useRoute();
const router = useRouter();
const campaign = ref<any>({});
const results = ref<any[]>([]);
const allTimeline = ref<any[]>([]);
const loadingResults = ref(false);
const resultChart = ref<HTMLElement | null>(null);
const drawerVisible = ref(false);
const drawerEmail = ref('');
const drawerEvents = ref<any[]>([]);

const resultColumns = [
  { title: '姓名', dataIndex: 'full_name', key: 'full_name' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '发件邮箱', dataIndex: 'smtp_from_address', key: 'smtp_from_address' },
  { title: '发送时间', dataIndex: 'send_date', key: 'send_date' },
  { title: '修改时间', dataIndex: 'modified_date', key: 'modified_date' },
  { title: '已报告', dataIndex: 'reported', key: 'reported' },
];

onMounted(async () => {
  const id = Number(route.params.id);
  await loadCampaign(id);
  await loadResults(id);
  initChart();
});

async function loadCampaign(id: number) {
  try {
    const data = await getCampaign(id);
    campaign.value = data;
    allTimeline.value = data.timeline || [];
  } catch (error) {
    message.error('加载活动详情失败');
  }
}

async function loadResults(id: number) {
  loadingResults.value = true;
  try {
    const data = await getCampaignResults(id);
    results.value = data.results || [];
    // Merge timeline from results API (may have more events)
    if (data.timeline?.length) {
      allTimeline.value = data.timeline;
    }
  } catch (error) {
    message.error('加载结果失败');
  } finally {
    loadingResults.value = false;
  }
}

function showRecipientTimeline(record: any) {
  drawerEmail.value = record.email;
  drawerEvents.value = allTimeline.value
    .filter((e: any) => e.email === record.email)
    .sort((a: any, b: any) => new Date(a.time).getTime() - new Date(b.time).getTime());
  drawerVisible.value = true;
}

function exportCSV(scope: string) {
  let data: any[];
  let filename: string;
  if (scope === 'results') {
    data = results.value;
    filename = `${campaign.value.name} - 结果.csv`;
  } else {
    data = allTimeline.value;
    filename = `${campaign.value.name} - 事件.csv`;
  }
  if (!data || data.length === 0) {
    message.warning('没有数据可导出');
    return;
  }
  const keys = Object.keys(data[0]);
  const csvRows = [keys.join(',')];
  for (const row of data) {
    csvRows.push(keys.map(k => {
      const val = row[k];
      if (val === null || val === undefined) return '';
      const str = String(val);
      if (str.includes(',') || str.includes('"') || str.includes('\n')) {
        return `"${str.replace(/"/g, '""')}"`;
      }
      return str;
    }).join(','));
  }
  const csvContent = csvRows.join('\r\n');
  const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.setAttribute('download', filename);
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

function initChart() {
  if (!resultChart.value) return;
  
  const chart = echarts.init(resultChart.value);
  const stats = campaign.value.stats || {};
  
  const option = {
    tooltip: { trigger: 'item' },
    legend: { orient: 'vertical', left: 'left' },
    series: [
      {
        name: '活动结果',
        type: 'pie',
        radius: '50%',
        data: [
          { value: stats.sent || 0, name: '已发送' },
          { value: stats.opened || 0, name: '已打开' },
          { value: stats.clicked || 0, name: '已点击' },
          { value: stats.submitted_data || 0, name: '已提交' },
          { value: stats.email_reported || 0, name: '已报告' },
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
      },
    ],
  };
  chart.setOption(option);
}

async function handleDuplicate() {
  router.push(`/campaigns?duplicate=${campaign.value.id}`);
}

async function handleLaunch() {
  Modal.confirm({
    title: '立即启动',
    content: `确定要立即启动活动「${campaign.value.name}」吗？`,
    onOk: async () => {
      try {
        await launchCampaign(campaign.value.id);
        message.success('活动已启动');
        loadCampaign(campaign.value.id);
      } catch (error) {
        message.error('启动失败');
      }
    },
  });
}

async function handleComplete() {
  Modal.confirm({
    title: '确认完成',
    content: '确定要完成这个活动吗？这将停止所有待发送的邮件。',
    onOk: async () => {
      try {
        await completeCampaign(campaign.value.id);
        message.success('活动已完成');
        loadCampaign(campaign.value.id);
      } catch (error) {
        message.error('操作失败');
      }
    },
  });
}

function getStatusColor(status: string) {
  const colors: Record<string, string> = {
    Completed: 'green',
    'In progress': 'blue',
    Queued: 'orange',
    Scheduled: 'cyan',
    Sending: 'purple',
  };
  return colors[status] || 'default';
}

function getStatusText(status: string) {
  const texts: Record<string, string> = {
    Completed: '已完成',
    'In progress': '进行中',
    Queued: '队列中',
    Scheduled: '已计划',
    Sending: '发送中',
  };
  return texts[status] || status;
}

function getResultStatusText(status: string) {
  const texts: Record<string, string> = {
    'Email Sent': '已发送',
    'Email Opened': '已打开',
    'Clicked Link': '已点击',
    'Submitted Data': '已提交',
    'Email Reported': '已报告',
    'Error Sending Email': '发送失败',
    Success: '已提交',
    Error: '错误',
    Unknown: '未知',
  };
  return texts[status] || status;
}

function getEventText(message: string) {
  const texts: Record<string, string> = {
    'Email Sent': '已发送',
    'Emails Sent': '已发送',
    'Email Opened': '已打开',
    'Clicked Link': '已点击',
    'Submitted Data': '已提交',
    'Email Reported': '已报告',
    'Error Sending Email': '发送失败',
    'Campaign Created': '活动已创建',
  };
  for (const [key, val] of Object.entries(texts)) {
    if (message.includes(key)) return val;
  }
  return message;
}

function getResultStatusColor(status: string) {
  const colors: Record<string, string> = {
    'Email Sent': 'blue',
    'Email Opened': 'cyan',
    'Clicked Link': 'orange',
    'Submitted Data': 'red',
    'Email Reported': 'green',
  };
  return colors[status] || 'default';
}

function getTimelineColor(message: string) {
  if (message.includes('Sent')) return 'blue';
  if (message.includes('Opened')) return 'cyan';
  if (message.includes('Clicked')) return 'orange';
  if (message.includes('Submitted')) return 'red';
  if (message.includes('Reported')) return 'green';
  if (message.includes('Error')) return 'red';
  return 'gray';
}

function getEventClass(message: string): string {
  if (message.includes('Sent')) return 'event-sent';
  if (message.includes('Opened')) return 'event-opened';
  if (message.includes('Clicked')) return 'event-clicked';
  if (message.includes('Submitted')) return 'event-submitted';
  if (message.includes('Reported')) return 'event-reported';
  if (message.includes('Error')) return 'event-error';
  return '';
}

function parseDetails(details: string): { key: string; value: string }[] | null {
  try {
    const obj = JSON.parse(details);
    if (typeof obj !== 'object' || obj === null) return null;
    
    const result: { key: string; value: string }[] = [];
    
    for (const [key, value] of Object.entries(obj)) {
      if (key === 'browser' && typeof value === 'object' && value !== null) {
        for (const [bk, bv] of Object.entries(value as Record<string, string>)) {
          result.push({ key: bk, value: String(bv) });
        }
      } else if (key === 'payload' && typeof value === 'object' && value !== null) {
        for (const [pk, pv] of Object.entries(value as Record<string, string>)) {
          result.push({ key: pk, value: String(pv) });
        }
      } else if (typeof value === 'object' && value !== null) {
        result.push({ key, value: JSON.stringify(value) });
      } else {
        result.push({ key, value: String(value ?? '') });
      }
    }
    
    return result.length > 0 ? result : null;
  } catch {
    return null;
  }
}
</script>

<style scoped>
.campaign-detail-container {
  background: #f0f2f5;
  min-height: 100vh;
}
.clickable-row {
  cursor: pointer;
}
.clickable-row:hover td {
  background: #e6f7ff !important;
}
.event-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 10px 12px;
  margin-bottom: 4px;
}
.event-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0;
}
.event-type {
  font-weight: 600;
  font-size: 13px;
  padding: 2px 8px;
  border-radius: 4px;
  background: #f0f0f0;
}
.event-sent { background: #e6f7ff; color: #1890ff; }
.event-opened { background: #e6fffb; color: #13c2c2; }
.event-clicked { background: #fff7e6; color: #fa8c16; }
.event-submitted { background: #fff2f0; color: #ff4d4f; }
.event-reported { background: #f6ffed; color: #52c41a; }
.event-error { background: #fff2f0; color: #ff4d4f; }
.event-time {
  color: #999;
  font-size: 12px;
}
.timeline-details {
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  padding: 8px 10px;
  margin-top: 8px;
}
.detail-item + .detail-item {
  border-top: 1px dashed #e8e8e8;
  padding-top: 6px;
  margin-top: 6px;
}
.detail-key {
  color: #888;
  font-size: 11px;
  font-weight: 500;
  margin-bottom: 2px;
}
.detail-value {
  color: #333;
  word-break: break-all;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12px;
  line-height: 1.5;
}
.detail-truncate {
  max-height: 60px;
  overflow-y: auto;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.detail-truncate:hover {
  -webkit-line-clamp: unset;
  max-height: 200px;
}
.detail-raw {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  color: #333;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12px;
}
</style>

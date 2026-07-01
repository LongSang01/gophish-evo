<template>
  <div class="dashboard-container">
    <a-row :gutter="16">
      <a-col v-for="s in statCharts" :key="s.key" :span="4">
        <a-card size="small" :body-style="{ padding: '16px' }">
          <div class="stat-card-body">
            <div class="stat-icon-wrap" :style="{ background: statsColorMapping[s.key] + '18' }">
              <svg v-if="s.key==='sent'" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 2L11 13"/><path d="M22 2l-7 20-4-9-9-4 20-7z"/></svg>
              <svg v-else-if="s.key==='opened'" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7z"/><circle cx="12" cy="12" r="3"/></svg>
              <svg v-else-if="s.key==='clicked'" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 3l14 8-7 2-3 7-4-14z"/></svg>
              <svg v-else-if="s.key==='submitted_data'" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
              <svg v-else-if="s.key==='email_reported'" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>
            </div>
            <div class="stat-info">
              <div class="stat-label">{{ s.label }}</div>
              <div class="stat-count">{{ s.count }}</div>
            </div>
          </div>
          <div :ref="(el) => setChartRef(el, s.key)" class="stat-chart"></div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="24">
        <a-card title="钓鱼成功率概览">
          <div ref="timelineChart" style="height: 300px"></div>
        </a-card>
      </a-col>
    </a-row>

    <a-row style="margin-top: 16px">
      <a-col :span="24">
        <a-card title="最近活动">
          <a-table
            :columns="columns"
            :data-source="campaigns"
            :loading="loading"
            :pagination="{ pageSize: 10 }"
            row-key="id"
            size="small"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'created_date'">
                {{ formatDate(record.created_date) }}
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">{{ getStatusText(record.status) }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button size="small" @click="viewDetail(record.id)">详情</a-button>
                  <a-button size="small" danger @click="handleDelete(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { message, Modal } from 'ant-design-vue';
import * as echarts from 'echarts';
import { getCampaignSummaries, deleteCampaign } from '@/api/campaigns';

const router = useRouter();
const loading = ref(false);
const campaigns = ref<any[]>([]);
const timelineChart = ref<HTMLElement | null>(null);
const chartRefs = reactive<Record<string, HTMLElement | null>>({});

const statsMapping: Record<string, string> = {
  sent: '已发送',
  opened: '已打开',
  email_reported: '已报告',
  clicked: '已点击',
  submitted_data: '已提交',
};

const statsColorMapping: Record<string, string> = {
  sent: '#1abc9c',
  opened: '#f9bf3b',
  email_reported: '#45d6ef',
  clicked: '#F39C12',
  submitted_data: '#f05b4f',
};

const statCharts = ref<{ key: string; label: string; count: number }[]>([]);

function setChartRef(el: any, key: string) {
  chartRefs[key] = el as HTMLElement;
}

const columns = [
  { title: '活动名称', dataIndex: 'name', key: 'name' },
  { title: '创建时间', dataIndex: 'created_date', key: 'created_date' },
  { title: '已发送', dataIndex: ['stats', 'sent'], key: 'sent' },
  { title: '已打开', dataIndex: ['stats', 'opened'], key: 'opened' },
  { title: '已点击', dataIndex: ['stats', 'clicked'], key: 'clicked' },
  { title: '已提交', dataIndex: ['stats', 'submitted_data'], key: 'submitted_data' },
  { title: '已报告', dataIndex: ['stats', 'email_reported'], key: 'email_reported' },
  { title: '状态', key: 'status' },
  { title: '操作', key: 'action' },
];

onMounted(async () => {
  await loadData();
});

async function loadData() {
  loading.value = true;
  try {
    const result = await getCampaignSummaries();
    const allCampaigns: any[] = result.campaigns || [];
    campaigns.value = allCampaigns.sort(
      (a: any, b: any) => new Date(b.created_date || 0).getTime() - new Date(a.created_date || 0).getTime()
    );
    if (allCampaigns.length > 0) {
      generateStatCharts(allCampaigns);
      generateTimelineChart(allCampaigns);
    }
  } catch (error) {
    message.error('加载数据失败');
  } finally {
    loading.value = false;
  }
}

function generateStatCharts(allCampaigns: any[]) {
  const agg: Record<string, number> = { sent: 0, opened: 0, clicked: 0, submitted_data: 0, email_reported: 0 };
  let total = 0;
  for (const c of allCampaigns) {
    const s = c.stats || {};
    for (const key of Object.keys(agg)) {
      agg[key] += s[key] || 0;
    }
    total += s.total || 0;
  }
  statCharts.value = Object.keys(agg).map(key => ({
    key,
    label: statsMapping[key] || key,
    count: agg[key],
  }));
  // Render donut charts on next tick
  setTimeout(() => {
    for (const key of Object.keys(agg)) {
      const el = chartRefs[key];
      if (!el) continue;
      const count = agg[key];
      const chart = echarts.init(el);
      chart.setOption({
        tooltip: { trigger: 'item', formatter: `{b}: {c} ({d}%)` },
        series: [{
          type: 'pie',
          radius: ['70%', '90%'],
          center: ['50%', '50%'],
          avoidLabelOverlap: false,
          label: { show: false },
          data: [
            { value: count, name: statsMapping[key] || key, itemStyle: { color: statsColorMapping[key] || '#ddd' } },
            { value: total - count, name: '', itemStyle: { color: '#dddddd' } },
          ],
        }],
      });
    }
  }, 100);
}

function generateTimelineChart(allCampaigns: any[]) {
  if (!timelineChart.value) return;
  const sorted = [...allCampaigns].sort(
    (a, b) => new Date(a.created_date).getTime() - new Date(b.created_date).getTime()
  );
  const data = sorted.map(c => {
    const stats = c.stats || {};
    const total = stats.total || 1;
    const clickRate = Math.floor(((stats.clicked || 0) / total) * 100);
    return {
      value: [new Date(c.created_date).getTime(), clickRate],
      campaign_id: c.id,
      name: c.name,
    };
  });
  const chart = echarts.init(timelineChart.value);
  chart.setOption({
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const p = params[0];
        if (!p) return '';
        return new Date(p.value[0]).toLocaleString('zh-CN') + '<br/>' +
          p.data.name + '<br/>成功率：<b>' + p.value[1] + '%</b>';
      },
    },
    xAxis: { type: 'time', name: '时间' },
    yAxis: { type: 'value', min: 0, max: 100, name: '成功率 %' },
    series: [{
      type: 'line',
      data: data,
      smooth: true,
      areaStyle: { opacity: 0.3 },
      lineStyle: { color: '#f05b4f' },
      itemStyle: { color: '#f05b4f' },
      symbol: 'circle',
      symbolSize: 6,
    }],
  });
  chart.on('click', (params: any) => {
    if (params.data?.campaign_id) {
      router.push(`/campaigns/${params.data.campaign_id}`);
    }
  });
}

function viewDetail(id: number) {
  router.push(`/campaigns/${id}`);
}

function handleDelete(campaign: any) {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除活动 "${campaign.name}" 吗？`,
    onOk: async () => {
      try {
        await deleteCampaign(campaign.id);
        message.success('删除成功');
        loadData();
      } catch (error) {
        message.error('删除失败');
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

function formatDate(date: string) {
  if (!date || date === '0001-01-01T00:00:00Z') return '-';
  return new Date(date).toLocaleString('zh-CN');
}
</script>

<style scoped>
.dashboard-container {
  padding: 24px;
}

.stat-chart {
  height: 80px;
  width: 100%;
  margin-top: 8px;
}

.stat-card-body {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stat-icon-wrap {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #555;
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-label {
  font-size: 12px;
  color: #888;
  margin-bottom: 2px;
}

.stat-count {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}
</style>
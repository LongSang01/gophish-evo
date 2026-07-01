<template>
  <div class="campaigns-container">
    <a-card>
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <PlusOutlined /> 新建活动
        </a-button>
      </template>
      
      <a-table
        :columns="columns"
        :data-source="campaigns"
        :loading="loading"
        :pagination="{ pageSize: 10 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'sent'">
            <a-tag color="#1abc9c">{{ record.stats?.sent ?? 0 }}</a-tag>
          </template>
          <template v-if="column.key === 'opened'">
            <a-tag color="#f9bf3b">{{ record.stats?.opened ?? 0 }}</a-tag>
          </template>
          <template v-if="column.key === 'clicked'">
            <a-tag color="#F39C12">{{ record.stats?.clicked ?? 0 }}</a-tag>
          </template>
          <template v-if="column.key === 'submitted_data'">
            <a-tag color="#f05b4f">{{ record.stats?.submitted_data ?? 0 }}</a-tag>
          </template>
          <template v-if="column.key === 'email_reported'">
            <a-tag color="#45d6ef">{{ record.stats?.email_reported ?? 0 }}</a-tag>
          </template>
          <template v-if="column.key === 'created_date'">
            {{ formatDate(record.created_date) }}
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="viewDetail(record.id)">详情</a-button>
              <a-button size="small" @click="handleCopyCampaign(record)">复制</a-button>
              <a-button size="small" danger @click="handleDelete(record.id)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="createModalVisible"
      title="新建钓鱼活动"
      @ok="handleCreate"
      :confirm-loading="creating"
      width="800px"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="活动名称" required>
          <a-input v-model:value="formData.name" placeholder="输入活动名称" />
        </a-form-item>
        <a-form-item label="模板" required>
          <a-select v-model:value="formData.template_id" placeholder="选择邮件模板">
            <a-select-option v-for="t in templates" :key="t.id" :value="t.id">
              {{ t.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="用户组" required>
          <a-select v-model:value="formData.group_ids" mode="multiple" placeholder="选择目标用户组">
            <a-select-option v-for="g in groups" :key="g.id" :value="g.id">
              {{ g.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="发送配置" required>
          <a-select v-model:value="formData.smtp_ids" mode="multiple" placeholder="选择一个或多个SMTP配置">
            <a-select-option v-for="s in smtpProfiles" :key="s.id" :value="s.id">
              {{ s.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="落地页">
          <a-select v-model:value="formData.page_id" placeholder="选择落地页（可选）">
            <a-select-option v-for="p in pages" :key="p.id" :value="p.id">
              {{ p.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="钓鱼URL">
          <a-input v-model:value="formData.url" placeholder="http://your-server.com" />
        </a-form-item>
        <a-form-item label="启动时间">
          <a-date-picker
            v-model:value="formData.launch_date"
            show-time
            format="YYYY-MM-DD HH:mm"
            placeholder="立即启动（留空）"
            style="width: 100%"
          />
        </a-form-item>
        <a-form-item label="发送截止时间">
          <a-date-picker
            v-model:value="formData.send_by_date"
            show-time
            format="YYYY-MM-DD HH:mm"
            placeholder="不限制（留空）"
            style="width: 100%"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { message, Modal } from 'ant-design-vue';
import { PlusOutlined } from '@ant-design/icons-vue';
import dayjs from 'dayjs';
import { getCampaignSummaries, getCampaign, createCampaign, deleteCampaign } from '@/api/campaigns';
import { getTemplates } from '@/api/templates';
import { getGroups } from '@/api/groups';
import { getSMTPProfiles } from '@/api/smtp';
import { getPages } from '@/api/pages';

const router = useRouter();
const route = useRoute();
const loading = ref(false);
const creating = ref(false);
const campaigns = ref<any[]>([]);
const templates = ref<any[]>([]);
const groups = ref<any[]>([]);
const smtpProfiles = ref<any[]>([]);
const pages = ref<any[]>([]);
const createModalVisible = ref(false);
const isDuplicate = ref(false);

const formData = ref({
  name: '',
  template_id: null as number | null,
  group_ids: [] as number[],
  smtp_ids: [] as number[],
  page_id: null as number | null,
  url: '',
  launch_date: null as any,
  send_by_date: null as any,
});

function buildCampaignPayload() {
  const template = templates.value.find((t: any) => t.id === formData.value.template_id);
  const smtps = formData.value.smtp_ids
    .map((id: number) => smtpProfiles.value.find((s: any) => s.id === id))
    .filter(Boolean)
    .map((s: any) => ({ name: s.name }));
  const page = pages.value.find((p: any) => p.id === formData.value.page_id);
  const selectedGroups = groups.value
    .filter((g: any) => formData.value.group_ids.includes(g.id))
    .map((g: any) => ({ name: g.name }));

  const payload: any = {
    name: formData.value.name,
    url: formData.value.url,
    template: template ? { name: template.name } : null,
    smtps: smtps.length > 0 ? smtps : undefined,
    page: page ? { name: page.name } : null,
    groups: selectedGroups,
  };

  if (formData.value.launch_date) {
    payload.launch_date = dayjs(formData.value.launch_date).toISOString();
  }
  if (formData.value.send_by_date) {
    payload.send_by_date = dayjs(formData.value.send_by_date).toISOString();
  }

  return payload;
}

const columns = [
  { title: '活动名称', dataIndex: 'name', key: 'name' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '发送数', dataIndex: ['stats', 'sent'], key: 'sent' },
  { title: '打开数', dataIndex: ['stats', 'opened'], key: 'opened' },
  { title: '点击数', dataIndex: ['stats', 'clicked'], key: 'clicked' },
  { title: '提交数', dataIndex: ['stats', 'submitted_data'], key: 'submitted_data' },
  { title: '报告数', dataIndex: ['stats', 'email_reported'], key: 'email_reported' },
  { title: '创建时间', dataIndex: 'created_date', key: 'created_date' },
  { title: '操作', key: 'action' },
];

onMounted(async () => {
  await loadData();
  const duplicateId = route.query.duplicate;
  if (duplicateId) {
    handleDuplicateFromDetail(Number(duplicateId));
  }
});

async function handleDuplicateFromDetail(id: number) {
  try {
    const campaign = await getCampaign(id);
    formData.value = {
      name: `${campaign.name} - 副本`,
      template_id: campaign.template?.id || null,
      group_ids: campaign.groups?.map((g: any) => g.id) || [],
      smtp_ids: campaign.smtps?.map((s: any) => s.id) || [],
      page_id: campaign.page?.id || null,
      url: campaign.url || '',
      launch_date: campaign.launch_date ? dayjs(campaign.launch_date) : null,
      send_by_date: campaign.send_by_date ? dayjs(campaign.send_by_date) : null,
    };
    isDuplicate.value = true;
    createModalVisible.value = true;
    router.replace({ path: '/campaigns' });
  } catch (error) {
    message.error('加载活动数据失败');
  }
}

async function loadData() {
  loading.value = true;
  try {
    const results = await Promise.allSettled([
      getCampaignSummaries(),
      getTemplates(),
      getGroups(),
      getSMTPProfiles(),
      getPages(),
    ]);
    const summaryResult = results[0].status === 'fulfilled' ? results[0].value : { campaigns: [] };
    campaigns.value = (summaryResult.campaigns || []).sort(
      (a: any, b: any) => new Date(b.created_date || 0).getTime() - new Date(a.created_date || 0).getTime()
    );
    templates.value = results[1].status === 'fulfilled' ? results[1].value : [];
    groups.value = results[2].status === 'fulfilled' ? results[2].value : [];
    smtpProfiles.value = results[3].status === 'fulfilled' ? results[3].value : [];
    pages.value = results[4].status === 'fulfilled' ? results[4].value : [];
  } catch (error) {
    message.error('加载数据失败');
  } finally {
    loading.value = false;
  }
}

function showCreateModal() {
  formData.value = {
    name: '',
    template_id: null,
    group_ids: [],
    smtp_ids: [],
    page_id: null,
    url: '',
    launch_date: null,
    send_by_date: null,
  };
  isDuplicate.value = false;
  createModalVisible.value = true;
}

async function handleCopyCampaign(campaign: any) {
  try {
    const fullCampaign = await getCampaign(campaign.id);
    formData.value = {
      name: `${fullCampaign.name} - 副本`,
      template_id: fullCampaign.template?.id || null,
      group_ids: fullCampaign.groups?.map((g: any) => g.id) || [],
      smtp_ids: fullCampaign.smtps?.map((s: any) => s.id) || [],
      page_id: fullCampaign.page?.id || null,
      url: fullCampaign.url || '',
      launch_date: fullCampaign.launch_date ? dayjs(fullCampaign.launch_date) : null,
      send_by_date: fullCampaign.send_by_date ? dayjs(fullCampaign.send_by_date) : null,
    };
    isDuplicate.value = true;
    createModalVisible.value = true;
  } catch (error) {
    message.error('加载活动数据失败');
  }
}

async function handleCreate() {
  creating.value = true;
  try {
    const payload = buildCampaignPayload();
    await createCampaign(payload);
    message.success(isDuplicate.value ? '活动复制成功' : '活动创建成功');
    createModalVisible.value = false;
    loadData();
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '创建失败');
  } finally {
    creating.value = false;
  }
}

function viewDetail(id: number) {
  router.push(`/campaigns/${id}`);
}

function handleDelete(id: number) {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除这个活动吗？此操作不可恢复。',
    onOk: async () => {
      try {
        await deleteCampaign(id);
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
.campaigns-container {
  padding: 24px;
}
</style>
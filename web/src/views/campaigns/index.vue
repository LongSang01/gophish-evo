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
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'source_type'">
            <a-tag :color="sourceTypeColor(record.source_type)">{{
              sourceTypeText(record.source_type)
            }}</a-tag>
          </template>
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
            <a-tag color="#f05b4f">{{
              record.stats?.submitted_data ?? 0
            }}</a-tag>
          </template>
          <template v-if="column.key === 'email_reported'">
            <a-tag color="#45d6ef">{{
              record.stats?.email_reported ?? 0
            }}</a-tag>
          </template>
          <template v-if="column.key === 'created_date'">
            {{ formatDate(record.created_date) }}
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="viewDetail(record.id)"
                >详情</a-button
              >
              <a-button size="small" @click="handleCopyCampaign(record)"
                >复制</a-button
              >
              <a-button size="small" danger @click="handleDelete(record.id)"
                >删除</a-button
              >
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
        <a-form-item label="活动类型">
          <a-radio-group
            v-model:value="formData.source_type"
            button-style="solid"
          >
            <a-radio-button value="email">邮件钓鱼</a-radio-button>
            <a-radio-button value="client">钓鱼客户端</a-radio-button>
            <a-radio-button value="page">固定钓鱼页面</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="活动名称" required>
          <a-input v-model:value="formData.name" placeholder="输入活动名称" />
        </a-form-item>

        <template v-if="formData.source_type === 'email'">
          <a-form-item label="模板" required>
            <a-select
              v-model:value="formData.template_id"
              placeholder="选择邮件模板"
            >
              <a-select-option v-for="t in templates" :key="t.id" :value="t.id">
                {{ t.name }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="用户组" required>
            <a-select
              v-model:value="formData.group_ids"
              mode="multiple"
              placeholder="选择目标用户组"
            >
              <a-select-option v-for="g in groups" :key="g.id" :value="g.id">
                {{ g.name }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="发送配置" required>
            <a-select
              v-model:value="formData.smtp_ids"
              mode="multiple"
              placeholder="选择一个或多个SMTP配置"
            >
              <a-select-option
                v-for="s in smtpProfiles"
                :key="s.id"
                :value="s.id"
              >
                {{ s.name }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="落地页">
            <a-select
              v-model:value="formData.page_id"
              placeholder="选择落地页（可选）"
            >
              <a-select-option v-for="p in pages" :key="p.id" :value="p.id">
                {{ p.name }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="钓鱼URL">
            <a-input
              v-model:value="formData.url"
              placeholder="http://phish_server"
            />
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
        </template>

        <template v-if="formData.source_type === 'client'">
          <a-form-item
            label="钓鱼URL"
            required
            extra="客户端数据将上报到 {地址}/api/report"
          >
            <a-input
              v-model:value="formData.url"
              placeholder="http://phish_server"
            />
          </a-form-item>
          <a-form-item label="去重主键">
            <a-select
              v-model:value="formData.report_config.dedup_key"
              placeholder="选择去重字段"
            >
              <a-select-option
                v-for="f in formData.report_config.fields"
                :key="f.key"
                :value="f.key"
                >{{ f.label || f.key }}</a-select-option
              >
              <a-select-option value="">不去重（每条都存储）</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="采集字段">
            <div
              style="
                display: flex;
                gap: 8px;
                margin-bottom: 8px;
                font-size: 12px;
                color: #999;
                padding: 0 2px;
              "
            >
              <span style="width: 100px">数据key</span>
              <span style="width: 100px">前端显示名</span>
              <span style="width: 110px">采集方式</span>
            </div>
            <div
              v-for="(f, idx) in formData.report_config.fields"
              :key="idx"
              style="
                display: flex;
                gap: 8px;
                margin-bottom: 8px;
                align-items: center;
              "
            >
              <a-input
                v-model:value="f.key"
                placeholder="key"
                style="width: 100px"
              />
              <a-input
                v-model:value="f.label"
                placeholder="显示名"
                style="width: 100px"
              />
              <a-select v-model:value="f.type" style="width: 110px">
                <a-select-option value="ip">ip</a-select-option>
                <a-select-option value="mac">mac</a-select-option>
                <a-select-option value="username">username</a-select-option>
                <a-select-option value="hostname">hostname</a-select-option>
                <a-select-option value="custom">custom</a-select-option>
              </a-select>
              <a-button
                type="text"
                danger
                size="small"
                @click="removeClientField(idx)"
                >删除</a-button
              >
            </div>
            <a-button type="dashed" block @click="addClientField"
              >+ 添加字段</a-button
            >
          </a-form-item>
        </template>

        <template v-if="formData.source_type === 'page'">
          <a-form-item label="落地页" required>
            <a-select v-model:value="formData.page_id" placeholder="选择落地页">
              <a-select-option v-for="p in pages" :key="p.id" :value="p.id">
                {{ p.name }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="钓鱼URL" required>
            <a-input
              v-model:value="formData.url"
              placeholder="http://phish_server"
            />
          </a-form-item>
          <a-form-item
            label="页面路径"
            required
            extra="页面访问路径，如 /login，与钓鱼URL拼接为完整地址"
          >
            <a-input v-model:value="formData.page_path" placeholder="/login" />
          </a-form-item>
        </template>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { message, Modal } from "ant-design-vue";
import { PlusOutlined } from "@ant-design/icons-vue";
import dayjs from "dayjs";
import {
  getCampaignSummaries,
  getCampaign,
  createCampaign,
  deleteCampaign,
} from "@/api/campaigns";
import { getTemplates } from "@/api/templates";
import { getGroups } from "@/api/groups";
import { getSMTPProfiles } from "@/api/smtp";
import { getPages } from "@/api/pages";
import {
  sourceTypeText,
  sourceTypeColor,
  defaultCampaignForm,
  fillFormFromCampaign,
} from "@/utils/campaign";

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
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
});

const formData = ref(defaultCampaignForm());

function buildCampaignPayload() {
  const template = templates.value.find(
    (t: any) => t.id === formData.value.template_id,
  );
  const smtps = formData.value.smtp_ids
    .map((id: number) => smtpProfiles.value.find((s: any) => s.id === id))
    .filter(Boolean)
    .map((s: any) => ({ name: s.name }));
  const page = pages.value.find((p: any) => p.id === formData.value.page_id);
  const selectedGroups = groups.value
    .filter((g: any) => formData.value.group_ids.includes(g.id))
    .map((g: any) => ({ name: g.name }));

  const payload: any = {
    source_type: formData.value.source_type,
    name: formData.value.name,
    url: formData.value.url,
  };

  if (formData.value.source_type === "client") {
    const fields = formData.value.report_config.fields
      .filter((f: any) => f.key && f.label)
      .map((f: any) => ({
        key: f.key,
        label: f.label,
        type: f.type || "text",
        required: !!f.required,
        placeholder: f.placeholder || "",
      }));
    payload.report_config = {
      fields,
      dedup_key: formData.value.report_config.dedup_key || "",
    };
    return payload;
  }

  if (formData.value.source_type === "page") {
    payload.url =
      (formData.value.url || "").replace(/\/+$/, "") +
      "/" +
      (formData.value.page_path || "").replace(/^\/+/, "");
    payload.page = page ? { name: page.name } : null;
    payload.report_config = {
      fields: [],
      dedup_key: "",
    };
    return payload;
  }

  // email type
  payload.template = template ? { name: template.name } : null;
  payload.smtps = smtps.length > 0 ? smtps : undefined;
  payload.page = page ? { name: page.name } : null;
  payload.groups = selectedGroups;

  if (formData.value.launch_date) {
    payload.launch_date = dayjs(formData.value.launch_date).toISOString();
  }
  if (formData.value.send_by_date) {
    payload.send_by_date = dayjs(formData.value.send_by_date).toISOString();
  }

  return payload;
}

function addClientField() {
  formData.value.report_config.fields.push({
    key: "",
    label: "",
    type: "custom",
    required: false,
    placeholder: "",
  });
}

function removeClientField(idx: number | string) {
  formData.value.report_config.fields.splice(Number(idx), 1);
}

const columns = [
  { title: "活动名称", dataIndex: "name", key: "name" },
  { title: "类型", dataIndex: "source_type", key: "source_type" },
  { title: "状态", dataIndex: "status", key: "status" },
  { title: "发送数", dataIndex: ["stats", "sent"], key: "sent" },
  { title: "打开数", dataIndex: ["stats", "opened"], key: "opened" },
  { title: "点击数", dataIndex: ["stats", "clicked"], key: "clicked" },
  {
    title: "提交数",
    dataIndex: ["stats", "submitted_data"],
    key: "submitted_data",
  },
  {
    title: "报告数",
    dataIndex: ["stats", "email_reported"],
    key: "email_reported",
  },
  { title: "创建时间", dataIndex: "created_date", key: "created_date" },
  { title: "操作", key: "action" },
];

onMounted(async () => {
  await loadCampaigns();
  await loadDropdownData();
  const duplicateId = route.query.duplicate;
  if (duplicateId) {
    handleDuplicateFromDetail(Number(duplicateId));
  }
});

async function handleDuplicateFromDetail(id: number) {
  try {
    const campaign = await getCampaign(id);
    fillDuplicateForm(campaign);
    isDuplicate.value = true;
    createModalVisible.value = true;
    router.replace({ path: "/campaigns" });
  } catch (error) {
    message.error("加载活动数据失败");
  }
}

async function loadCampaigns() {
  loading.value = true;
  try {
    const result = await getCampaignSummaries({
      pageNum: pagination.current,
      pageSize: pagination.pageSize,
    });
    campaigns.value = result.data || result.campaigns || [];
    pagination.total = result.total || 0;
  } catch (error) {
    message.error("加载活动列表失败");
  } finally {
    loading.value = false;
  }
}

async function loadDropdownData() {
  const results = await Promise.allSettled([
    getTemplates(),
    getGroups(),
    getSMTPProfiles(),
    getPages(),
  ]);
  templates.value = results[0].status === "fulfilled" ? results[0].value : [];
  groups.value = results[1].status === "fulfilled" ? results[1].value : [];
  smtpProfiles.value =
    results[2].status === "fulfilled" ? results[2].value : [];
  pages.value = results[3].status === "fulfilled" ? results[3].value : [];
}

function handleTableChange(pag: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  loadCampaigns();
}

function resetPagination() {
  pagination.current = 1;
  loadCampaigns();
}

function showCreateModal() {
  formData.value = defaultCampaignForm();
  isDuplicate.value = false;
  createModalVisible.value = true;
}

function fillDuplicateForm(campaign: any) {
  formData.value = defaultCampaignForm();
  fillFormFromCampaign(formData.value, campaign, dayjs);
}

async function handleCopyCampaign(campaign: any) {
  try {
    const fullCampaign = await getCampaign(campaign.id);
    fillDuplicateForm(fullCampaign);
    isDuplicate.value = true;
    createModalVisible.value = true;
  } catch (error) {
    message.error("加载活动数据失败");
  }
}

async function handleCreate() {
  creating.value = true;
  try {
    const payload = buildCampaignPayload();
    await createCampaign(payload);
    message.success(isDuplicate.value ? "活动复制成功" : "活动创建成功");
    createModalVisible.value = false;
    resetPagination();
  } catch (error: any) {
    message.error(
      error?.response?.data?.message || error?.message || "创建失败",
    );
  } finally {
    creating.value = false;
  }
}

function viewDetail(id: number) {
  router.push(`/campaigns/${id}`);
}

function handleDelete(id: number) {
  Modal.confirm({
    title: "确认删除",
    content: "确定要删除这个活动吗？此操作不可恢复。",
    onOk: async () => {
      try {
        await deleteCampaign(id);
        message.success("删除成功");
        resetPagination();
      } catch (error) {
        message.error("删除失败");
      }
    },
  });
}

function getStatusColor(status: string) {
  const colors: Record<string, string> = {
    Completed: "green",
    "In progress": "blue",
    Queued: "orange",
    Scheduled: "cyan",
    Sending: "purple",
  };
  return colors[status] || "default";
}

function getStatusText(status: string) {
  const texts: Record<string, string> = {
    Completed: "已完成",
    "In progress": "进行中",
    Queued: "队列中",
    Scheduled: "已计划",
    Sending: "发送中",
  };
  return texts[status] || status;
}

function formatDate(date: string) {
  if (!date || date === "0001-01-01T00:00:00Z") return "-";
  return new Date(date).toLocaleString("zh-CN");
}
</script>

<style scoped>
.campaigns-container {
  padding: 24px;
}
</style>

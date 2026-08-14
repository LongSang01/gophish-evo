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
          <a-button
            v-if="campaign.source_type === 'client'"
            type="primary"
            @click="showClientCode"
          >
            <CodeOutlined /> 生成客户端代码
          </a-button>
          <a-button
            v-if="campaign.source_type === 'page'"
            type="primary"
            @click="showPageUrl"
          >
            <LinkOutlined /> 固定页面URL
          </a-button>
          <a-button @click="handleDuplicate">
            <CopyOutlined /> 复制配置
          </a-button>
          <a-button
            v-if="
              campaign.status === 'In progress' || campaign.status === 'Queued'
            "
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
        <a-card v-if="campaign.source_type === 'email'" title="活动统计">
          <a-row :gutter="16">
            <a-col :span="8">
              <a-statistic
                title="邮件已发送"
                :value="campaign.stats?.sent || 0"
              />
            </a-col>
            <a-col :span="8">
              <a-statistic
                title="邮件已打开"
                :value="campaign.stats?.opened || 0"
              />
            </a-col>
            <a-col :span="8">
              <a-statistic
                title="链接已点击"
                :value="campaign.stats?.clicked || 0"
              />
            </a-col>
          </a-row>
          <div ref="resultChart" style="height: 300px; margin-top: 24px"></div>
        </a-card>
        <a-card v-else title="上报统计">
          <a-statistic
            title="上报数量"
            :value="reportsTotal"
            :value-style="{ fontSize: '48px' }"
          />
        </a-card>
      </a-col>

      <a-col :span="8">
        <a-card title="活动详情">
          <a-descriptions :column="1" size="small">
            <a-descriptions-item label="活动类型">
              <a-tag :color="sourceTypeColor(campaign.source_type)">{{
                sourceTypeText(campaign.source_type)
              }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="创建时间">
              {{ formatDate(campaign.created_date) }}
            </a-descriptions-item>
            <a-descriptions-item
              v-if="campaign.source_type === 'email'"
              label="启动时间"
            >
              {{ formatDate(campaign.launch_date) }}
            </a-descriptions-item>
            <template v-if="campaign.source_type === 'email'">
              <a-descriptions-item label="发送配置">
                {{ campaign.smtps?.[0]?.name }}
              </a-descriptions-item>
              <a-descriptions-item label="邮件模板">
                {{ campaign.template?.name }}
              </a-descriptions-item>
              <a-descriptions-item label="用户组">
                <a-tag v-for="g in campaign.groups" :key="g.id">{{
                  g.name
                }}</a-tag>
                <span v-if="!campaign.groups?.length">无</span>
              </a-descriptions-item>
              <a-descriptions-item label="发送数">
                <a-statistic
                  :value="campaign.stats?.sent || 0"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
              <a-descriptions-item label="打开数">
                <a-statistic
                  :value="campaign.stats?.opened || 0"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
              <a-descriptions-item label="点击数">
                <a-statistic
                  :value="campaign.stats?.clicked || 0"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
              <a-descriptions-item label="提交数">
                <a-statistic
                  :value="campaign.stats?.submitted_data || 0"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
              <a-descriptions-item label="报告数">
                <a-statistic
                  :value="campaign.stats?.email_reported || 0"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
            </template>
            <template v-else-if="campaign.source_type === 'client'">
              <a-descriptions-item label="去重主键">
                {{ campaign.report_config?.dedup_key || "-" }}
              </a-descriptions-item>
              <a-descriptions-item label="字段配置">
                {{
                  (campaign.report_config?.fields || [])
                    .map((f: any) => `${f.label}(${f.key})`)
                    .join("、") || "-"
                }}
              </a-descriptions-item>
              <a-descriptions-item label="上报数量">
                <a-statistic
                  :value="reportsTotal"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
            </template>
            <template v-else>
              <a-descriptions-item label="打开数">
                <a-statistic
                  :value="campaign.stats?.opened || 0"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
              <a-descriptions-item label="点击数">
                <a-statistic
                  :value="campaign.stats?.clicked || 0"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
              <a-descriptions-item label="提交数">
                <a-statistic
                  :value="campaign.stats?.submitted_data || 0"
                  :value-style="{ fontSize: '16px' }"
                />
              </a-descriptions-item>
            </template>
          </a-descriptions>
        </a-card>
      </a-col>
    </a-row>

    <a-card
      v-if="campaign.source_type === 'email'"
      title="用户结果"
      style="margin: 16px 24px"
    >
      <template #extra>
        <a-space>
          <a-button size="small" @click="exportCSV('results')"
            >导出结果CSV</a-button
          >
          <a-button size="small" @click="exportCSV('events')"
            >导出事件CSV</a-button
          >
        </a-space>
      </template>
      <a-table
        :columns="resultColumns"
        :data-source="results"
        :loading="loadingResults"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
        :custom-row="
          (record: any) => ({ onClick: () => showRecipientTimeline(record) })
        "
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
              {{ record.reported ? "是" : "否" }}
            </a-tag>
          </template>
          <template v-if="column.key === 'send_date'">
            {{ formatDate(record.send_date) }}
          </template>
          <template v-if="column.key === 'smtp_from_address'">
            {{ record.smtp_from_address || "-" }}
          </template>
          <template v-if="column.key === 'modified_date'">
            {{ formatDate(record.modified_date) }}
          </template>
        </template>
      </a-table>
    </a-card>

    <a-card v-else title="上报数据" style="margin: 16px 24px">
      <template #extra>
        <a-space>
          <a-button size="small" @click="exportReports">导出上报CSV</a-button>
        </a-space>
      </template>
      <a-table
        :columns="reportColumns"
        :data-source="reports"
        :loading="loadingReports"
        :pagination="reportPagination"
        @change="handleReportTableChange"
        row-key="id"
        :custom-row="
          (record: any) => ({ onClick: () => showReportDetails(record) })
        "
        :row-class-name="() => 'clickable-row'"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'source'">
            <a-tag>{{ record.source === "client" ? "客户端" : "页面" }}</a-tag>
          </template>
          <template v-if="column.key === 'data_preview'">
            <div
              v-if="record.data && Object.keys(record.data).length > 0"
              class="preview-cell"
            >
              <a-tooltip placement="left">
                <template #title>
                  <div
                    v-for="(v, k) in record.data"
                    :key="k"
                    style="margin-bottom: 2px"
                  >
                    <b>{{ k }}:</b> {{ v }}
                  </div>
                </template>
                <span class="preview-chips">
                  <a-tag
                    v-for="(v, k) in record.data"
                    :key="k"
                    class="preview-chip"
                  >
                    {{ v }}
                  </a-tag>
                </span>
              </a-tooltip>
            </div>
            <span v-else style="color: #bbb">—</span>
          </template>
          <template v-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template
            v-else-if="column.dataIndex && column.dataIndex.startsWith('data.')"
          >
            {{ getDataField(record, column.dataIndex.slice(5)) }}
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
              <span class="event-type" :class="getEventClass(event.message)">{{
                getEventText(event.message)
              }}</span>
              <span class="event-time">{{ formatDate(event.time) }}</span>
            </div>
            <div v-if="event.details" class="timeline-details">
              <template v-if="parseDetails(event.details)">
                <div
                  v-for="(item, idx) in parseDetails(event.details)"
                  :key="idx"
                  class="detail-item"
                >
                  <div class="detail-key">{{ item.key }}</div>
                  <div
                    class="detail-value"
                    :class="{ 'detail-truncate': item.value.length > 80 }"
                    :title="item.value"
                  >
                    {{ item.value }}
                  </div>
                </div>
              </template>
              <pre v-else class="detail-raw">{{ event.details }}</pre>
            </div>
          </div>
        </a-timeline-item>
        <a-empty v-if="!drawerEvents.length" description="暂无事件" />
      </a-timeline>
    </a-drawer>

    <a-modal
      v-model:open="clientCodeVisible"
      title="钓鱼客户端源码"
      :footer="null"
      width="780px"
    >
      <a-alert
        type="info"
        show-icon
        style="margin-bottom: 12px"
        message="复制以下代码，在任意机器上执行 go build 后投放"
      />
      <a-alert type="warning" show-icon style="margin-bottom: 12px">
        <template #message>
          <div>推荐编译选项（隐藏命令行窗口 + 去除调试信息 + 减小体积）：</div>
          <code style="word-break: break-all">
            GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H
            windowsgui" -o main.exe main.go
          </code>
        </template>
      </a-alert>
      <a-space style="margin-bottom: 12px">
        <a-button size="small" type="primary" @click="copyClientCode"
          >复制全部代码</a-button
        >
        <a-button size="small" @click="downloadClientCode"
          >下载 main.go</a-button
        >
      </a-space>
      <a-textarea
        :value="clientCode"
        :rows="22"
        readonly
        style="
          font-family: monospace;
          font-size: 12px;
          background: #1e1e1e;
          color: #d4d4d4;
        "
      />
    </a-modal>

    <a-modal
      v-model:open="pageUrlVisible"
      title="固定钓鱼页面 URL"
      :footer="null"
      width="560px"
    >
      <a-alert
        type="info"
        show-icon
        style="margin-bottom: 12px"
        message="可自行扩展至其他钓鱼场景"
      />
      <a-input :value="pageFullUrl" readonly style="margin-bottom: 12px">
        <template #addonAfter>
          <a-button type="link" size="small" @click="copyPageUrl"
            >复制</a-button
          >
        </template>
      </a-input>
      <div v-if="pageQrCode" style="text-align: center; margin-top: 16px">
        <img
          :src="pageQrCode"
          alt="QR Code"
          style="width: 200px; height: 200px"
        />
        <div style="color: #999; font-size: 12px; margin-top: 8px">
          扫码访问钓鱼页面
        </div>
      </div>
    </a-modal>

    <a-drawer
      v-model:open="reportDrawerVisible"
      :title="`上报详情 - ${reportSourceLabel}`"
      width="500"
    >
      <a-descriptions
        v-if="reportDetail"
        :column="1"
        size="small"
        style="margin-bottom: 16px"
      >
        <a-descriptions-item label="来源">
          <a-tag>{{ reportSourceLabel }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="IP">{{
          reportDetail.ip || "-"
        }}</a-descriptions-item>
        <template
          v-if="campaign.source_type === 'page' && reportDetail.user_agent"
        >
          <a-descriptions-item label="User-Agent">{{
            reportDetail.user_agent
          }}</a-descriptions-item>
        </template>
        <a-descriptions-item label="上报时间">{{
          formatDate(reportDetail.created_at)
        }}</a-descriptions-item>
      </a-descriptions>
      <a-timeline>
        <a-timeline-item color="blue">
          <div class="event-card">
            <div class="event-header">
              <span class="event-type">采集数据</span>
              <span class="event-time">{{
                formatDate(reportDetail?.created_at)
              }}</span>
            </div>
            <div v-if="reportDetailRows.length" class="timeline-details">
              <div
                v-for="(item, idx) in reportDetailRows"
                :key="idx"
                class="detail-item"
              >
                <div class="detail-key">{{ item.key }}</div>
                <div
                  class="detail-value"
                  :class="{ 'detail-truncate': item.value.length > 80 }"
                  :title="item.value"
                >
                  {{ item.value }}
                </div>
              </div>
            </div>
            <a-empty v-else description="无采集字段" />
          </div>
        </a-timeline-item>
      </a-timeline>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { message, Modal } from "ant-design-vue";
import {
  CopyOutlined,
  StopOutlined,
  PlayCircleOutlined,
  CodeOutlined,
  LinkOutlined,
} from "@ant-design/icons-vue";
import * as echarts from "echarts";
import {
  getCampaign,
  getCampaignResults,
  completeCampaign,
  launchCampaign,
  getClientCode,
  getPageURL,
  getCampaignReports,
  getCampaignReportSummary,
  exportCampaignReports,
} from "@/api/campaigns";
import QRCode from "qrcode";
import { formatDate } from "@/utils/format";
import { sourceTypeText, sourceTypeColor } from "@/utils/campaign";

const route = useRoute();
const router = useRouter();
const campaign = ref<any>({});
const results = ref<any[]>([]);
const allTimeline = ref<any[]>([]);
const loadingResults = ref(false);
const resultChart = ref<HTMLElement | null>(null);
const drawerVisible = ref(false);
const drawerEmail = ref("");
const drawerEvents = ref<any[]>([]);
const clientCodeVisible = ref(false);
const clientCode = ref("");
const pageUrlVisible = ref(false);
const pageFullUrl = ref("");
const pageQrCode = ref("");
const reports = ref<any[]>([]);
const reportsTotal = ref(0);
const loadingReports = ref(false);
const reportPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
});
const reportColumns = ref<any[]>([]);
const reportDrawerVisible = ref(false);
const reportDetail = ref<any>(null);
const reportDetailRows = ref<{ key: string; value: string }[]>([]);
const reportSourceLabel = computed(() =>
  reportDetail.value?.source === "client" ? "客户端" : "页面",
);
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
});

const resultColumns = [
  { title: "姓名", dataIndex: "full_name", key: "full_name" },
  { title: "邮箱", dataIndex: "email", key: "email" },
  { title: "状态", dataIndex: "status", key: "status" },
  {
    title: "发件邮箱",
    dataIndex: "smtp_from_address",
    key: "smtp_from_address",
  },
  { title: "发送时间", dataIndex: "send_date", key: "send_date" },
  { title: "修改时间", dataIndex: "modified_date", key: "modified_date" },
  { title: "已报告", dataIndex: "reported", key: "reported" },
];

onMounted(async () => {
  const id = Number(route.params.id);
  await loadCampaign(id);
  if (campaign.value.source_type === "email") {
    await loadResults(id);
    initChart();
  } else {
    await loadReports(id);
  }
});

async function loadCampaign(id: number) {
  try {
    const data = await getCampaign(id);
    campaign.value = data;
    allTimeline.value = data.timeline || [];
    if (data.source_type === "client" || data.source_type === "page") {
      buildReportColumns();
    }
  } catch (error) {
    message.error("加载活动详情失败");
  }
}

async function loadResults(id: number) {
  loadingResults.value = true;
  try {
    const data = await getCampaignResults(id, {
      pageNum: pagination.current,
      pageSize: pagination.pageSize,
    });
    results.value = data.results || [];
    pagination.total = data.total || 0;
    // Merge timeline from results API (may have more events)
    if (data.timeline?.length) {
      allTimeline.value = data.timeline;
    }
  } catch (error) {
    message.error("加载结果失败");
  } finally {
    loadingResults.value = false;
  }
}

function handleTableChange(pag: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  loadResults(Number(route.params.id));
}

function buildReportColumns() {
  const cols: any[] = [
    { title: "来源", dataIndex: "source", key: "source", width: 90 },
    { title: "IP", dataIndex: "ip", key: "ip", width: 150, ellipsis: true },
  ];

  // Page-type campaigns show click count and submission status in summary mode.
  if (campaign.value.source_type === "page") {
    cols.push({
      title: "提交次数",
      dataIndex: "submission_count",
      key: "submission_count",
      width: 100,
      customRender: ({ record }: any) =>
        record.submitted ? record.submission_count : 0,
    });
    cols.push({
      title: "点击次数",
      dataIndex: "click_count",
      key: "click_count",
      width: 100,
    });
  }

  cols.push({
    title: "数据预览",
    dataIndex: "data",
    key: "data_preview",
    width: campaign.value.source_type === "page" ? 300 : 450,
    ellipsis: true,
  });
  cols.push({
    title: "时间",
    dataIndex: "created_at",
    key: "created_at",
    width: 180,
    customRender: ({ record }: any) => {
      // Summary rows use last_seen_at; regular reports use created_at.
      const ts =
        record.created_at || record.last_seen_at || record.last_click_at;
      return ts ? formatDate(ts) : "—";
    },
  });

  reportColumns.value = cols;
}

async function loadReports(id: number) {
  loadingReports.value = true;
  try {
    // For page-type campaigns, use the summary API that merges click stats
    // with submitted reports.
    const isPage = campaign.value.source_type === "page";
    const apiFn = isPage ? getCampaignReportSummary : getCampaignReports;
    const data = await apiFn(id, {
      pageNum: reportPagination.current,
      pageSize: reportPagination.pageSize,
    });
    reports.value = data.data || data || [];
    reportsTotal.value = data.total ?? reports.value.length;
    reportPagination.total = data.total ?? reports.value.length;
  } catch (error) {
    message.error("加载上报数据失败");
  } finally {
    loadingReports.value = false;
  }
}

function handleReportTableChange(pag: any) {
  reportPagination.current = pag.current;
  reportPagination.pageSize = pag.pageSize;
  loadReports(Number(route.params.id));
}

function getDataField(record: any, key: string) {
  const v = record.data?.[key];
  if (v === null || v === undefined || v === "") return "-";
  return String(v);
}

function showReportDetails(record: any) {
  reportDetail.value = record;
  const rows: { key: string; value: string }[] = [];
  const data = record.data || {};
  for (const [k, v] of Object.entries(data)) {
    rows.push({ key: k, value: String(v ?? "") });
  }
  // For summary rows, show click/submission stats as additional info.
  if (record.submitted !== undefined) {
    rows.push({ key: "提交次数", value: String(record.submission_count ?? 0) });
    rows.push({ key: "额外点击次数", value: String(record.click_count ?? 0) });
  }
  reportDetailRows.value = rows;
  reportDrawerVisible.value = true;
}

async function exportReports() {
  try {
    const blob: any = await exportCampaignReports(Number(route.params.id));
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `${campaign.value.name} - 上报.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  } catch (error) {
    message.error("导出失败");
  }
}

async function showClientCode() {
  try {
    const data = await getClientCode(Number(route.params.id));
    clientCode.value = typeof data === "string" ? data : data.data || "";
    clientCodeVisible.value = true;
  } catch (error) {
    message.error("获取客户端代码失败");
  }
}

async function copyClientCode() {
  try {
    await navigator.clipboard.writeText(clientCode.value);
    message.success("已复制");
  } catch {
    message.error("复制失败，请手动选择复制");
  }
}

function downloadClientCode() {
  const blob = new Blob([clientCode.value], {
    type: "text/plain;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.setAttribute("download", "main.go");
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

async function showPageUrl() {
  try {
    const data = await getPageURL(Number(route.params.id));
    pageFullUrl.value = typeof data === "string" ? data : data.data || "";
  } catch {
    pageFullUrl.value = campaign.value.url || "";
  }
  pageQrCode.value = "";
  if (pageFullUrl.value) {
    try {
      pageQrCode.value = await QRCode.toDataURL(pageFullUrl.value, {
        width: 256,
        margin: 2,
      });
    } catch {
      // QR generation is non-critical
    }
  }
  pageUrlVisible.value = true;
}

async function copyPageUrl() {
  try {
    await navigator.clipboard.writeText(pageFullUrl.value);
    message.success("已复制");
  } catch {
    message.error("复制失败");
  }
}

function showRecipientTimeline(record: any) {
  drawerEmail.value = record.email;
  drawerEvents.value = allTimeline.value
    .filter((e: any) => e.email === record.email)
    .sort(
      (a: any, b: any) =>
        new Date(a.time).getTime() - new Date(b.time).getTime(),
    );
  drawerVisible.value = true;
}

async function exportCSV(scope: string) {
  let data: any[];
  let filename: string;
  if (scope === "results") {
    try {
      const full = await getCampaignResults(Number(route.params.id));
      data = full.results || [];
    } catch {
      message.error("导出结果失败");
      return;
    }
    filename = `${campaign.value.name} - 结果.csv`;
  } else {
    data = allTimeline.value;
    filename = `${campaign.value.name} - 事件.csv`;
  }
  if (!data || data.length === 0) {
    message.warning("没有数据可导出");
    return;
  }
  const keys = Object.keys(data[0]);
  const csvRows = [keys.join(",")];
  for (const row of data) {
    csvRows.push(
      keys
        .map((k) => {
          const val = row[k];
          if (val === null || val === undefined) return "";
          const str = String(val);
          if (str.includes(",") || str.includes('"') || str.includes("\n")) {
            return `"${str.replace(/"/g, '""')}"`;
          }
          return str;
        })
        .join(","),
    );
  }
  const csvContent = csvRows.join("\r\n");
  const blob = new Blob(["\uFEFF" + csvContent], {
    type: "text/csv;charset=utf-8;",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.setAttribute("download", filename);
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
    tooltip: { trigger: "item" },
    legend: { orient: "vertical", left: "left" },
    series: [
      {
        name: "活动结果",
        type: "pie",
        radius: "50%",
        data: [
          { value: stats.sent || 0, name: "已发送" },
          { value: stats.opened || 0, name: "已打开" },
          { value: stats.clicked || 0, name: "已点击" },
          { value: stats.submitted_data || 0, name: "已提交" },
          { value: stats.email_reported || 0, name: "已报告" },
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: "rgba(0, 0, 0, 0.5)",
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
    title: "立即启动",
    content: `确定要立即启动活动「${campaign.value.name}」吗？`,
    onOk: async () => {
      try {
        await launchCampaign(campaign.value.id);
        message.success("活动已启动");
        loadCampaign(campaign.value.id);
      } catch (error) {
        message.error("启动失败");
      }
    },
  });
}

async function handleComplete() {
  Modal.confirm({
    title: "确认完成",
    content: "确定要完成这个活动吗？这将停止所有待发送的邮件。",
    onOk: async () => {
      try {
        await completeCampaign(campaign.value.id);
        message.success("活动已完成");
        loadCampaign(campaign.value.id);
      } catch (error) {
        message.error("操作失败");
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

function getResultStatusText(status: string) {
  const texts: Record<string, string> = {
    "Email Sent": "已发送",
    "Email Opened": "已打开",
    "Clicked Link": "已点击",
    "Submitted Data": "已提交",
    "Email Reported": "已报告",
    "Error Sending Email": "发送失败",
    Success: "已提交",
    Error: "错误",
    Unknown: "未知",
  };
  return texts[status] || status;
}

function getEventText(message: string) {
  const texts: Record<string, string> = {
    "Email Sent": "已发送",
    "Emails Sent": "已发送",
    "Email Opened": "已打开",
    "Clicked Link": "已点击",
    "Submitted Data": "已提交",
    "Email Reported": "已报告",
    "Error Sending Email": "发送失败",
    "Campaign Created": "活动已创建",
  };
  for (const [key, val] of Object.entries(texts)) {
    if (message.includes(key)) return val;
  }
  return message;
}

function getResultStatusColor(status: string) {
  const colors: Record<string, string> = {
    "Email Sent": "blue",
    "Email Opened": "cyan",
    "Clicked Link": "orange",
    "Submitted Data": "red",
    "Email Reported": "green",
  };
  return colors[status] || "default";
}

function getTimelineColor(message: string) {
  if (message.includes("Sent")) return "blue";
  if (message.includes("Opened")) return "cyan";
  if (message.includes("Clicked")) return "orange";
  if (message.includes("Submitted")) return "red";
  if (message.includes("Reported")) return "green";
  if (message.includes("Error")) return "red";
  return "gray";
}

function getEventClass(message: string): string {
  if (message.includes("Sent")) return "event-sent";
  if (message.includes("Opened")) return "event-opened";
  if (message.includes("Clicked")) return "event-clicked";
  if (message.includes("Submitted")) return "event-submitted";
  if (message.includes("Reported")) return "event-reported";
  if (message.includes("Error")) return "event-error";
  return "";
}

function parseDetails(
  details: string,
): { key: string; value: string }[] | null {
  try {
    const obj = JSON.parse(details);
    if (typeof obj !== "object" || obj === null) return null;

    const result: { key: string; value: string }[] = [];

    for (const [key, value] of Object.entries(obj)) {
      if (key === "browser" && typeof value === "object" && value !== null) {
        for (const [bk, bv] of Object.entries(
          value as Record<string, string>,
        )) {
          result.push({ key: bk, value: String(bv) });
        }
      } else if (
        key === "payload" &&
        typeof value === "object" &&
        value !== null
      ) {
        for (const [pk, pv] of Object.entries(
          value as Record<string, string>,
        )) {
          result.push({ key: pk, value: String(pv) });
        }
      } else if (typeof value === "object" && value !== null) {
        result.push({ key, value: JSON.stringify(value) });
      } else {
        result.push({ key, value: String(value ?? "") });
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
.preview-cell {
  max-width: 100%;
}
.preview-chips {
  display: inline-flex;
  gap: 6px;
  flex-wrap: wrap;
  overflow: hidden;
  max-width: 100%;
}
.preview-chip {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  line-height: 22px;
  margin-inline-end: 0 !important;
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
.event-sent {
  background: #e6f7ff;
  color: #1890ff;
}
.event-opened {
  background: #e6fffb;
  color: #13c2c2;
}
.event-clicked {
  background: #fff7e6;
  color: #fa8c16;
}
.event-submitted {
  background: #fff2f0;
  color: #ff4d4f;
}
.event-reported {
  background: #f6ffed;
  color: #52c41a;
}
.event-error {
  background: #fff2f0;
  color: #ff4d4f;
}
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
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
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
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  font-size: 12px;
}
</style>

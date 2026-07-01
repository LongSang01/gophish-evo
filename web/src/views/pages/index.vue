<template>
  <div class="pages-container">
    <a-card>
      <template #extra>
        <a-space>
          <a-button @click="showImportSiteModal">
            <ImportOutlined /> 导入站点
          </a-button>
          <a-button type="primary" @click="showCreateModal">
            <PlusOutlined /> 新建落地页
          </a-button>
        </a-space>
      </template>
      
      <a-table
        :columns="columns"
        :data-source="pages"
        :loading="loading"
        :pagination="{ pageSize: 10 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'capture_credentials'">
            <a-tag :color="record.capture_credentials ? 'green' : 'default'">
              {{ record.capture_credentials ? '是' : '否' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'modified_date'">
            {{ formatDate(record.modified_date) }}
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="showEditModal(record)">编辑</a-button>
              <a-button size="small" @click="handleDuplicate(record)">复制</a-button>
              <a-button size="small" danger @click="handleDelete(record.id)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="modalVisible"
      :title="editingPage ? '编辑落地页' : '新建落地页'"
      @ok="handleSave"
      :confirm-loading="saving"
      width="1000px"
      :footer="null"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="页面名称" required>
          <a-input v-model:value="formData.name" placeholder="输入页面名称" />
        </a-form-item>
        <a-form-item label="HTML内容">
          <div class="editor-preview-wrapper">
            <div class="editor-pane">
              <div class="pane-header">
                <span>代码编辑</span>
                <a-button size="small" @click="togglePreview">
                  {{ showPreview ? '隐藏预览' : '显示预览' }}
                </a-button>
              </div>
              <Codemirror
                v-if="modalVisible"
                v-model="formData.html"
                :style="{ height: '380px' }"
                :extensions="cmExtensions"
                :key="editingPage ? editingPage.id : 'new'"
                :tab-size="2"
                :autofocus="true"
                placeholder="输入页面 HTML 内容..."
              />
            </div>
            <div v-if="showPreview" class="preview-pane">
              <div class="pane-header">实时预览</div>
              <iframe
                ref="previewFrame"
                class="preview-iframe"
                :srcdoc="formData.html"
                sandbox="allow-same-origin"
              />
            </div>
          </div>
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="formData.capture_credentials">
            捕获凭证
          </a-checkbox>
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="formData.capture_passwords">
            捕获密码
          </a-checkbox>
        </a-form-item>
        <a-form-item label="重定向URL">
          <a-input v-model:value="formData.redirect_url" placeholder="提交后重定向到的URL（可选）" />
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="handleSave" :loading="saving">
              保存
            </a-button>
            <a-button @click="modalVisible = false">取消</a-button>
          </a-space>
        </a-form-item>
      </a-form>
      <a-modal
        v-model:open="importSiteVisible"
        title="导入站点"
        @ok="handleImportSite"
        :confirm-loading="importingSite"
        width="600px"
      >
        <a-form layout="vertical">
          <a-form-item label="站点URL" required>
            <a-input v-model:value="importSiteUrl" placeholder="https://example.com/login" />
          </a-form-item>
        </a-form>
      </a-modal>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { PlusOutlined, ImportOutlined } from '@ant-design/icons-vue';
import { Codemirror } from 'vue-codemirror';
import { html } from '@codemirror/lang-html';
import { oneDark } from '@codemirror/theme-one-dark';
import { getPages, createPage, updatePage, deletePage, importSite } from '@/api/pages';
import { formatDate } from '@/utils/format';

const loading = ref(false);
const saving = ref(false);
const pages = ref<any[]>([]);
const modalVisible = ref(false);
const editingPage = ref<any>(null);

const formData = ref({
  name: '',
  html: '',
  capture_credentials: false,
  capture_passwords: false,
  redirect_url: '',
});

const cmExtensions = [html(), oneDark];
const showPreview = ref(false);
const previewFrame = ref<HTMLIFrameElement | null>(null);

const importSiteVisible = ref(false);
const importSiteUrl = ref('');
const importingSite = ref(false);

function togglePreview() {
  showPreview.value = !showPreview.value;
}

const columns = [
  { title: '页面名称', dataIndex: 'name', key: 'name' },
  { title: '捕获凭证', dataIndex: 'capture_credentials', key: 'capture_credentials' },
  { title: '修改时间', dataIndex: 'modified_date', key: 'modified_date' },
  { title: '操作', key: 'action' },
];

onMounted(() => {
  loadPages();
});

async function loadPages() {
  loading.value = true;
  try {
    pages.value = (await getPages()).sort(
      (a: any, b: any) => new Date(b.modified_date).getTime() - new Date(a.modified_date).getTime()
    );
  } catch (error) {
    message.error('加载落地页失败');
  } finally {
    loading.value = false;
  }
}

function showCreateModal() {
  editingPage.value = null;
  formData.value = {
    name: '',
    html: '',
    capture_credentials: false,
    capture_passwords: false,
    redirect_url: '',
  };
  modalVisible.value = true;
}

function showEditModal(page: any) {
  editingPage.value = page;
  formData.value = {
    name: page.name,
    html: page.html || '',
    capture_credentials: page.capture_credentials || false,
    capture_passwords: page.capture_passwords || false,
    redirect_url: page.redirect_url || '',
  };
  modalVisible.value = true;
}

function handleDuplicate(page: any) {
  editingPage.value = null;
  formData.value = {
    name: `${page.name} (副本)`,
    html: page.html || '',
    capture_credentials: page.capture_credentials || false,
    capture_passwords: page.capture_passwords || false,
    redirect_url: page.redirect_url || '',
  };
  modalVisible.value = true;
}

async function handleSave() {
  saving.value = true;
  try {
    if (editingPage.value) {
      await updatePage(editingPage.value.id, formData.value);
      message.success('落地页更新成功');
    } else {
      await createPage(formData.value);
      message.success('落地页创建成功');
    }
    modalVisible.value = false;
    loadPages();
  } catch (error: any) {
    message.error(error?.message || '保存失败');
  } finally {
    saving.value = false;
  }
}

function showImportSiteModal() {
  importSiteUrl.value = '';
  importSiteVisible.value = true;
}

async function handleImportSite() {
  if (!importSiteUrl.value) {
    message.warning('请输入站点URL');
    return;
  }
  importingSite.value = true;
  try {
    const data = await importSite({ url: importSiteUrl.value });
    formData.value.html = data.html;
    importSiteVisible.value = false;
    message.success('站点导入成功');
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '导入失败');
  } finally {
    importingSite.value = false;
  }
}

function handleDelete(id: number) {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除这个落地页吗？此操作不可恢复。',
    onOk: async () => {
      try {
        await deletePage(id);
        message.success('删除成功');
        loadPages();
      } catch (error) {
        message.error('删除失败');
      }
    },
  });
}
</script>

<style scoped>
.pages-container {
  padding: 24px;
}

.editor-preview-wrapper {
  display: flex;
  gap: 0;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  overflow: hidden;
}

.editor-pane {
  flex: 1;
  min-width: 0;
}

.preview-pane {
  flex: 1;
  min-width: 0;
  border-left: 1px solid #d9d9d9;
  background: #fff;
}

.pane-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 12px;
  background: #fafafa;
  border-bottom: 1px solid #e8e8e8;
  font-size: 12px;
  color: #666;
}

.editor-pane :deep(.cm-editor) {
  height: 380px;
}

.editor-pane :deep(.cm-scroller) {
  font-family: 'Fira Code', 'Consolas', 'Monaco', monospace;
}

.preview-iframe {
  width: 100%;
  height: 380px;
  border: none;
  background: #fff;
}
</style>

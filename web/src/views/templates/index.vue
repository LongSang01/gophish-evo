<template>
  <div class="templates-container">
    <a-card>
      <template #extra>
        <a-space>
          <a-button @click="showImportEmailModal">
            <ImportOutlined /> 导入邮件
          </a-button>
          <a-button type="primary" @click="showCreateModal">
            <PlusOutlined /> 新建模板
          </a-button>
        </a-space>
      </template>
      
      <a-table
        :columns="columns"
        :data-source="templates"
        :loading="loading"
        :pagination="{ pageSize: 10 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a-tag color="purple">{{ record.name }}</a-tag>
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
      :title="editingTemplate ? '编辑模板' : '新建模板'"
      @ok="handleSave"
      :confirm-loading="saving"
      width="1000px"
      :footer="null"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="模板名称" required>
          <a-input v-model:value="formData.name" placeholder="输入模板名称" />
        </a-form-item>
        <a-form-item label="邮件主题" required>
          <a-input v-model:value="formData.subject" placeholder="输入邮件主题" />
        </a-form-item>
        <a-form-item label="发件人">
          <a-input v-model:value="formData.envelope_sender" placeholder="发件人名称或地址" />
        </a-form-item>
        <a-form-item label="邮件内容 (HTML)">
          <div class="editor-preview-wrapper">
            <div class="editor-pane">
              <div class="pane-header">
                <span>代码编辑</span>
                <a-space>
                  <a-checkbox v-model:checked="formData.useTracker">添加追踪图片</a-checkbox>
                  <a-button size="small" @click="togglePreview">
                    {{ showPreview ? '隐藏预览' : '显示预览' }}
                  </a-button>
                </a-space>
              </div>
              <Codemirror
                v-if="modalVisible"
                v-model="formData.html"
                :style="{ height: '380px' }"
                :extensions="cmExtensions"
                :key="editorKey"
                :tab-size="2"
                :autofocus="true"
                placeholder="输入邮件 HTML 内容..."
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
        <a-form-item label="纯文本内容">
          <a-textarea v-model:value="formData.text" :rows="4" placeholder="纯文本版本（可选）" />
        </a-form-item>
        <a-form-item label="附件">
          <a-table
            :columns="attachmentColumns"
            :data-source="formData.attachments"
            :pagination="false"
            size="small"
            row-key="name"
          >
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.key === 'name'">
                <a-input v-model:value="record.name" size="small" placeholder="文件名" />
              </template>
              <template v-if="column.key === 'type'">
                <a-input v-model:value="record.type" size="small" placeholder="MIME类型" />
              </template>
              <template v-if="column.key === 'action'">
                <a-button size="small" danger @click="formData.attachments.splice(index, 1)">
                  <DeleteOutlined />
                </a-button>
              </template>
            </template>
          </a-table>
          <a-space style="margin-top: 8px">
            <a-upload
              :before-upload="handleAttachmentUpload"
              :show-upload-list="false"
              multiple
            >
              <a-button><UploadOutlined /> 上传附件</a-button>
            </a-upload>
          </a-space>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button @click="handlePreview">
              <EyeOutlined /> 预览
            </a-button>
            <a-button @click="handleTestSend">
              <SendOutlined /> 发送测试
            </a-button>
            <a-button type="primary" @click="handleSave" :loading="saving">
              保存
            </a-button>
            <a-button @click="modalVisible = false">取消</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="previewVisible"
      title="邮件预览"
      width="800px"
      :footer="null"
    >
      <div v-html="previewHtml" class="preview-content"></div>
    </a-modal>

    <a-modal
      v-model:open="testSendVisible"
      title="发送测试邮件"
      @ok="handleTestSendConfirm"
      :confirm-loading="testSending"
      width="500px"
    >
      <a-form layout="vertical">
        <a-form-item label="发送配置" required>
          <a-select v-model:value="testSendForm.smtp_name" placeholder="选择SMTP发送配置">
            <a-select-option v-for="s in smtpProfiles" :key="s.id" :value="s.name">
              {{ s.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="收件人邮箱" required>
          <a-input v-model:value="testSendForm.to" placeholder="输入测试收件人邮箱" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="importEmailVisible"
      title="导入邮件"
      @ok="handleImportEmail"
      :confirm-loading="importingEmail"
      width="700px"
    >
      <a-form layout="vertical">
        <a-form-item label="邮件原始内容" required>
          <a-textarea v-model:value="importEmailContent" :rows="12" placeholder="粘贴邮件原始内容（包含邮件头）..." />
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="importEmailConvertLinks">
             将链接转换为 <code>{{ convertLinksPlaceholder }}</code>
          </a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { PlusOutlined, EyeOutlined, SendOutlined, ImportOutlined, UploadOutlined, DeleteOutlined } from '@ant-design/icons-vue';
import { Codemirror } from 'vue-codemirror';
import { html } from '@codemirror/lang-html';
import { oneDark } from '@codemirror/theme-one-dark';
import { getTemplates, createTemplate, updateTemplate, deleteTemplate, sendTestEmail, importEmail } from '@/api/templates';
import { getSMTPProfiles } from '@/api/smtp';
import { formatDate } from '@/utils/format';

const loading = ref(false);
const saving = ref(false);
const templates = ref<any[]>([]);
const modalVisible = ref(false);
const previewVisible = ref(false);
const previewHtml = ref('');
const editingTemplate = ref<any>(null);
const editorKey = ref(0);

const attachmentColumns = [
  { title: '文件名', key: 'name' },
  { title: 'MIME类型', key: 'type' },
  { title: '操作', key: 'action', width: 60 },
];

const smtpProfiles = ref<any[]>([]);
const testSendVisible = ref(false);
const testSending = ref(false);
const testSendForm = ref({ smtp_name: '', to: '' });

const importEmailVisible = ref(false);
const importEmailContent = ref('');
const importEmailConvertLinks = ref(true);
const importingEmail = ref(false);
const convertLinksPlaceholder = '{{.URL}}';

const formData = ref({
  name: '',
  subject: '',
  envelope_sender: '',
  html: '',
  text: '',
  useTracker: true,
  attachments: [] as any[],
});

const cmExtensions = [html(), oneDark];
const showPreview = ref(false);
const previewFrame = ref<HTMLIFrameElement | null>(null);

function togglePreview() {
  showPreview.value = !showPreview.value;
}

const columns = [
  { title: '模板名称', dataIndex: 'name', key: 'name' },
  { title: '邮件主题', dataIndex: 'subject', key: 'subject' },
  { title: '修改时间', dataIndex: 'modified_date', key: 'modified_date' },
  { title: '操作', key: 'action' },
];

onMounted(() => {
  loadTemplates();
  loadSMTPProfiles();
});

async function loadTemplates() {
  loading.value = true;
  try {
    templates.value = (await getTemplates()).sort(
      (a: any, b: any) => new Date(b.modified_date).getTime() - new Date(a.modified_date).getTime()
    );
  } catch (error) {
    message.error('加载模板失败');
  } finally {
    loading.value = false;
  }
}

async function loadSMTPProfiles() {
  try {
    smtpProfiles.value = await getSMTPProfiles();
  } catch (error) {
    // silent
  }
}

function showCreateModal() {
  editingTemplate.value = null;
  formData.value = {
    name: '',
    subject: '',
    envelope_sender: '',
    html: '',
    text: '',
    useTracker: true,
    attachments: [],
  };
  editorKey.value = Date.now();
  modalVisible.value = true;
}

function showEditModal(template: any) {
  editingTemplate.value = template;
  const hasTracker = template.html && (template.html.includes('{{.Tracker}}') || template.html.includes('{{.TrackingUrl}}'));
  formData.value = {
    name: template.name,
    subject: template.subject,
    envelope_sender: template.envelope_sender || '',
    html: template.html || '',
    text: template.text || '',
    useTracker: hasTracker,
    attachments: template.attachments ? template.attachments.map((a: any) => ({ ...a })) : [],
  };
  editorKey.value = Date.now();
  modalVisible.value = true;
}

function handleDuplicate(template: any) {
  editingTemplate.value = null;
  const hasTracker = template.html && (template.html.includes('{{.Tracker}}') || template.html.includes('{{.TrackingUrl}}'));
  formData.value = {
    name: `${template.name} - 副本`,
    subject: template.subject,
    envelope_sender: template.envelope_sender || '',
    html: template.html || '',
    text: template.text || '',
    useTracker: hasTracker,
    attachments: template.attachments ? template.attachments.map((a: any) => ({ ...a })) : [],
  };
  editorKey.value = Date.now();
  modalVisible.value = true;
}

function handleAttachmentUpload(file: File) {
  const reader = new FileReader();
  reader.onload = (e: any) => {
    const content = e.target.result.split(',')[1];
    const type = file.type || 'application/octet-stream';
    formData.value.attachments.push({
      name: file.name,
      content,
      type,
    });
  };
  reader.readAsDataURL(file);
  return false;
}

async function handleSave() {
  saving.value = true;
  try {
    let html = formData.value.html;
    if (formData.value.useTracker) {
      if (html.indexOf('{{.Tracker}}') === -1 && html.indexOf('{{.TrackingUrl}}') === -1) {
        html = html.replace('</body>', '{{.Tracker}}</body>');
      }
    } else {
      html = html.replace('{{.Tracker}}</body>', '</body>');
    }
    const payload = {
      name: formData.value.name,
      subject: formData.value.subject,
      envelope_sender: formData.value.envelope_sender,
      html: html,
      text: formData.value.text,
      attachments: formData.value.attachments,
    };
    if (editingTemplate.value) {
      await updateTemplate(editingTemplate.value.id, payload);
      message.success('模板更新成功');
    } else {
      await createTemplate(payload);
      message.success('模板创建成功');
    }
    modalVisible.value = false;
    loadTemplates();
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '保存失败');
  } finally {
    saving.value = false;
  }
}

function handlePreview() {
  previewHtml.value = formData.value.html;
  previewVisible.value = true;
}

async function handleTestSend() {
  if (!formData.value.name) {
    message.warning('请先保存模板后再发送测试');
    return;
  }
  testSendForm.value = { smtp_name: '', to: '' };
  testSendVisible.value = true;
}

async function handleTestSendConfirm() {
  if (!testSendForm.value.smtp_name) {
    message.warning('请选择发送配置');
    return;
  }
  if (!testSendForm.value.to) {
    message.warning('请输入收件人邮箱');
    return;
  }
  testSending.value = true;
  try {
    await sendTestEmail({
      template: { name: formData.value.name },
      smtp: { name: testSendForm.value.smtp_name },
      email: testSendForm.value.to,
    });
    message.success('测试邮件发送成功');
    testSendVisible.value = false;
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '测试邮件发送失败');
  } finally {
    testSending.value = false;
  }
}

function handleDelete(id: number) {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除这个模板吗？此操作不可恢复。',
    onOk: async () => {
      try {
        await deleteTemplate(id);
        message.success('删除成功');
        loadTemplates();
      } catch (error) {
        message.error('删除失败');
      }
    },
  });
}

function showImportEmailModal() {
  importEmailContent.value = '';
  importEmailConvertLinks.value = true;
  importEmailVisible.value = true;
}

async function handleImportEmail() {
  if (!importEmailContent.value) {
    message.warning('请输入邮件内容');
    return;
  }
  importingEmail.value = true;
  try {
    const data = await importEmail({
      content: importEmailContent.value,
      convert_links: importEmailConvertLinks.value,
    });
    if (data.subject) formData.value.subject = data.subject;
    if (data.html) formData.value.html = data.html;
    if (data.text) formData.value.text = data.text;
    importEmailVisible.value = false;
    message.success('邮件导入成功');
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '导入失败');
  } finally {
    importingEmail.value = false;
  }
}
</script>

<style scoped>
.templates-container {
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

.preview-content {
  padding: 16px;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  background: #fff;
}
</style>
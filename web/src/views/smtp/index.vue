<template>
  <div class="smtp-container">
    <a-card>
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <PlusOutlined /> 新建发送配置
        </a-button>
      </template>
      
      <a-table
        :columns="columns"
        :data-source="profiles"
        :loading="loading"
        :pagination="{ pageSize: 10 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'host_display'">
            <code>{{ record.host }}</code>
          </template>
          <template v-if="column.key === 'modified_date'">
            {{ formatDate(record.modified_date) }}
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="showEditModal(record)">编辑</a-button>
              <a-button size="small" @click="handleDuplicate(record)">复制</a-button>
              <a-button size="small" @click="handleTest(record)">测试</a-button>
              <a-button size="small" danger @click="handleDelete(record.id)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="testSendVisible"
      title="发送测试邮件"
      @ok="handleTestSendConfirm"
      :confirm-loading="testSending"
      width="500px"
    >
      <a-form layout="vertical">
        <a-form-item label="收件人邮箱" required>
          <a-input v-model:value="testSendForm.to" placeholder="输入测试收件人邮箱" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="modalVisible"
      :title="editingProfile ? '编辑发送配置' : '新建发送配置'"
      @ok="handleSave"
      :confirm-loading="saving"
      width="700px"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="配置名称" required>
          <a-input v-model:value="formData.name" placeholder="输入配置名称" />
        </a-form-item>
        <a-form-item label="SMTP地址" required>
          <a-input v-model:value="formData.host" placeholder="smtp.example.com:587" />
        </a-form-item>
        <a-form-item label="用户名">
          <a-input v-model:value="formData.username" placeholder="SMTP用户名" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model:value="formData.password" placeholder="SMTP密码" />
        </a-form-item>
        <a-form-item label="发件人地址" required>
          <a-input v-model:value="formData.from_address" placeholder="sender@example.com" />
        </a-form-item>
        <a-form-item label="忽略证书错误">
          <a-checkbox v-model:checked="formData.ignore_cert_errors">
            忽略SSL/TLS证书错误
          </a-checkbox>
        </a-form-item>
        <a-form-item label="自定义头部">
          <a-table
            :columns="headerColumns"
            :data-source="formData.headers"
            :pagination="false"
            size="small"
            row-key="uid"
          >
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.key === 'key'">
                <a-input v-model:value="record.key" size="small" placeholder="头部名称" />
              </template>
              <template v-if="column.key === 'value'">
                <a-input v-model:value="record.value" size="small" placeholder="头部值" />
              </template>
              <template v-if="column.key === 'action'">
                <a-button size="small" danger @click="formData.headers.splice(index, 1)">
                  <DeleteOutlined />
                </a-button>
              </template>
            </template>
          </a-table>
          <a-space style="margin-top: 8px">
            <a-input v-model:value="newHeaderKey" size="small" placeholder="头部名称" style="width: 200px" />
            <a-input v-model:value="newHeaderValue" size="small" placeholder="头部值" style="width: 200px" />
            <a-button @click="addCustomHeader"><PlusOutlined /> 添加</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons-vue';

let headerUid = 0;
import { getSMTPProfiles, createSMTPProfile, updateSMTPProfile, deleteSMTPProfile, sendTestEmail } from '@/api/smtp';
import { formatDate } from '@/utils/format';

const loading = ref(false);
const saving = ref(false);
const profiles = ref<any[]>([]);
const modalVisible = ref(false);
const editingProfile = ref<any>(null);
const testSendVisible = ref(false);
const testSending = ref(false);
const testSendForm = ref({ smtp_name: '', to: '' });

const newHeaderKey = ref('');
const newHeaderValue = ref('');

const formData = ref({
  name: '',
  host: '',
  username: '',
  password: '',
  from_address: '',
  ignore_cert_errors: false,
  headers: [] as any[],
});

const headerColumns = [
  { title: '名称', key: 'key', width: 200 },
  { title: '值', key: 'value', width: 200 },
  { title: '操作', key: 'action', width: 60 },
];

const columns = [
  { title: '配置名称', dataIndex: 'name', key: 'name' },
  { title: 'SMTP地址', key: 'host_display' },
  { title: '发件人', dataIndex: 'from_address', key: 'from_address' },
  { title: '修改时间', dataIndex: 'modified_date', key: 'modified_date' },
  { title: '操作', key: 'action' },
];

onMounted(() => {
  loadProfiles();
});

async function loadProfiles() {
  loading.value = true;
  try {
    profiles.value = (await getSMTPProfiles()).sort(
      (a: any, b: any) => new Date(b.modified_date).getTime() - new Date(a.modified_date).getTime()
    );
  } catch (error) {
    message.error('加载发送配置失败');
  } finally {
    loading.value = false;
  }
}

function showCreateModal() {
  editingProfile.value = null;
  formData.value = {
    name: '',
    host: '',
    username: '',
    password: '',
    from_address: '',
    ignore_cert_errors: false,
    headers: [],
  };
  modalVisible.value = true;
}

function assignHeaderUids(headers: any[]) {
  headers.forEach(h => { h.uid = ++headerUid; });
  return headers;
}

function showEditModal(profile: any) {
  editingProfile.value = profile;
  formData.value = {
    name: profile.name,
    host: profile.host,
    username: profile.username || '',
    password: profile.password || '',
    from_address: profile.from_address || '',
    ignore_cert_errors: profile.ignore_cert_errors || false,
    headers: profile.headers ? assignHeaderUids(profile.headers.map((h: any) => ({ ...h }))) : [],
  };
  modalVisible.value = true;
}

function handleDuplicate(profile: any) {
  editingProfile.value = null;
  formData.value = {
    name: `${profile.name} - 副本`,
    host: profile.host,
    username: profile.username || '',
    password: profile.password || '',
    from_address: profile.from_address || '',
    ignore_cert_errors: profile.ignore_cert_errors || false,
    headers: profile.headers ? assignHeaderUids(profile.headers.map((h: any) => ({ ...h }))) : [],
  };
  modalVisible.value = true;
}

function addCustomHeader() {
  if (!newHeaderKey.value || !newHeaderValue.value) {
    message.warning('请填写头部名称和值');
    return;
  }
  formData.value.headers.push({
    uid: ++headerUid,
    key: newHeaderKey.value,
    value: newHeaderValue.value,
  });
  newHeaderKey.value = '';
  newHeaderValue.value = '';
}

async function handleSave() {
  saving.value = true;
  try {
    const payload = {
      ...formData.value,
    };
    if (editingProfile.value) {
      await updateSMTPProfile(editingProfile.value.id, payload);
      message.success('发送配置更新成功');
    } else {
      await createSMTPProfile(payload);
      message.success('发送配置创建成功');
    }
    modalVisible.value = false;
    loadProfiles();
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '保存失败');
  } finally {
    saving.value = false;
  }
}

function handleTest(profile: any) {
  testSendForm.value = { smtp_name: profile.name, to: '' };
  testSendVisible.value = true;
}

async function handleTestSendConfirm() {
  if (!testSendForm.value.to) {
    message.warning('请输入收件人邮箱');
    return;
  }
  testSending.value = true;
  try {
    await sendTestEmail({
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
    content: '确定要删除这个发送配置吗？此操作不可恢复。',
    onOk: async () => {
      try {
        await deleteSMTPProfile(id);
        message.success('删除成功');
        loadProfiles();
      } catch (error) {
        message.error('删除失败');
      }
    },
  });
}
</script>

<style scoped>
.smtp-container {
  padding: 24px;
}
</style>
<template>
  <div class="webhooks-container">
    <a-card>
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <PlusOutlined /> 新建Webhook
        </a-button>
      </template>
      
      <a-table
        :columns="columns"
        :data-source="webhooks"
        :loading="loading"
        :pagination="{ pageSize: 10 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'active'">
            <a-tag :color="record.is_active ? 'green' : 'default'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="showEditModal(record)">编辑</a-button>
              <a-button size="small" @click="handleValidate(record)">验证</a-button>
              <a-button size="small" danger @click="handleDelete(record.id)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="modalVisible"
      :title="editingWebhook ? '编辑Webhook' : '新建Webhook'"
      @ok="handleSave"
      :confirm-loading="saving"
      width="600px"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="formData.name" placeholder="输入Webhook名称" />
        </a-form-item>
        <a-form-item label="URL" required>
          <a-input v-model:value="formData.url" placeholder="https://example.com/webhook" />
        </a-form-item>
        <a-form-item label="密钥">
          <a-input v-model:value="formData.secret" placeholder="用于签名的密钥（可选）" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="formData.is_active" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { PlusOutlined } from '@ant-design/icons-vue';
import { getWebhooks, createWebhook, updateWebhook, deleteWebhook, validateWebhook } from '@/api/webhooks';

const loading = ref(false);
const saving = ref(false);
const webhooks = ref<any[]>([]);
const modalVisible = ref(false);
const editingWebhook = ref<any>(null);

const formData = ref({
  name: '',
  url: '',
  secret: '',
  is_active: true,
});

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: 'URL', dataIndex: 'url', key: 'url' },
  { title: '状态', dataIndex: 'is_active', key: 'active' },
  { title: '修改时间', dataIndex: 'modified_date', key: 'modified_date' },
  { title: '操作', key: 'action' },
];

onMounted(() => {
  loadWebhooks();
});

async function loadWebhooks() {
  loading.value = true;
  try {
    webhooks.value = (await getWebhooks()).sort(
      (a: any, b: any) => new Date(b.modified_date).getTime() - new Date(a.modified_date).getTime()
    );
  } catch (error) {
    message.error('加载Webhooks失败');
  } finally {
    loading.value = false;
  }
}

function showCreateModal() {
  editingWebhook.value = null;
  formData.value = {
    name: '',
    url: '',
    secret: '',
    is_active: true,
  };
  modalVisible.value = true;
}

function showEditModal(webhook: any) {
  editingWebhook.value = webhook;
  formData.value = {
    name: webhook.name,
    url: webhook.url,
    secret: webhook.secret || '',
    is_active: webhook.is_active !== false,
  };
  modalVisible.value = true;
}

async function handleSave() {
  saving.value = true;
  try {
    if (editingWebhook.value) {
      await updateWebhook(editingWebhook.value.id, formData.value);
      message.success('Webhook更新成功');
    } else {
      await createWebhook(formData.value);
      message.success('Webhook创建成功');
    }
    modalVisible.value = false;
    loadWebhooks();
  } catch (error: any) {
    message.error(error?.message || '保存失败');
  } finally {
    saving.value = false;
  }
}

async function handleValidate(webhook: any) {
  try {
    await validateWebhook(webhook.id);
    message.success('Webhook验证成功');
  } catch (error) {
    message.error('Webhook验证失败');
  }
}

function handleDelete(id: number) {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除这个Webhook吗？此操作不可恢复。',
    onOk: async () => {
      try {
        await deleteWebhook(id);
        message.success('删除成功');
        loadWebhooks();
      } catch (error) {
        message.error('删除失败');
      }
    },
  });
}
</script>

<style scoped>
.webhooks-container {
  padding: 24px;
}
</style>

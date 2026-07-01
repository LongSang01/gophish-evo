<template>
  <div class="users-container">
    <a-card>
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <PlusOutlined /> 新建用户
        </a-button>
      </template>
      
      <a-table
        :columns="columns"
        :data-source="users"
        :loading="loading"
        :pagination="{ pageSize: 10 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'username'">
            <a-tag :color="record.role?.slug === 'admin' ? 'red' : 'blue'">{{ record.username }}</a-tag>
          </template>
          <template v-if="column.key === 'role'">
            <a-tag :color="record.role?.slug === 'admin' ? 'red' : 'blue'">
              {{ record.role?.slug === 'admin' ? '管理员' : '用户' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'locked'">
            <a-tag :color="record.account_locked ? 'orange' : 'green'">
              {{ record.account_locked ? '已锁定' : '正常' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'last_login'">
            {{ formatDate(record.last_login) }}
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="handleImpersonate(record)">切换</a-button>
              <a-button size="small" @click="showEditModal(record)">编辑</a-button>
              <a-button size="small" danger @click="handleDelete(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="modalVisible"
      :title="editingUser ? '编辑用户' : '新建用户'"
      @ok="handleSave"
      :confirm-loading="saving"
      width="500px"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="用户名" required>
          <a-input v-model:value="formData.username" placeholder="输入用户名" :disabled="isAdminUser" />
        </a-form-item>
        <a-form-item v-if="!editingUser" label="密码" required>
          <a-input-password v-model:value="formData.password" placeholder="输入密码" />
        </a-form-item>
        <a-form-item label="角色" required>
          <a-select v-model:value="formData.role">
            <a-select-option value="admin">管理员</a-select-option>
            <a-select-option value="user">用户</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item v-if="!editingUser" label="首次登录修改密码">
          <a-checkbox v-model:checked="formData.password_change_required">
            要求用户首次登录修改密码
          </a-checkbox>
        </a-form-item>
        <a-form-item label="账户状态">
          <a-checkbox v-model:checked="formData.account_locked">
            锁定账户
          </a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { PlusOutlined } from '@ant-design/icons-vue';
import { getUsers, createUser, updateUser, deleteUser } from '@/api/users';
import { formatDate } from '@/utils/format';

const loading = ref(false);
const saving = ref(false);
const users = ref<any[]>([]);
const modalVisible = ref(false);
const editingUser = ref<any>(null);

const isAdminUser = computed(() => editingUser.value?.username === 'admin');

const formData = ref({
  username: '',
  password: '',
  role: 'user',
  password_change_required: false,
  account_locked: false,
});

const columns = [
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '角色', dataIndex: 'role', key: 'role' },
  { title: '状态', dataIndex: 'account_locked', key: 'locked' },
  { title: '最后登录', dataIndex: 'last_login', key: 'last_login' },
  { title: '操作', key: 'action' },
];

onMounted(() => {
  loadUsers();
});

async function loadUsers() {
  loading.value = true;
  try {
    users.value = (await getUsers()).sort(
      (a: any, b: any) => (b.id || 0) - (a.id || 0)
    );
  } catch (error) {
    message.error('加载用户列表失败');
  } finally {
    loading.value = false;
  }
}

function showCreateModal() {
  editingUser.value = null;
  formData.value = {
    username: '',
    password: '',
    role: 'user',
    password_change_required: false,
    account_locked: false,
  };
  modalVisible.value = true;
}

function showEditModal(user: any) {
  editingUser.value = user;
  formData.value = {
    username: user.username,
    password: '',
    role: user.role?.slug || 'user',
    password_change_required: user.password_change_required || false,
    account_locked: user.account_locked || false,
  };
  modalVisible.value = true;
}

function handleImpersonate(user: any) {
  Modal.confirm({
    title: '确认切换用户',
    content: `您将退出当前账户并以 ${user.username} 的身份登录`,
    onOk: async () => {
      try {
        const resp = await fetch('/impersonate', {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: `username=${encodeURIComponent(user.username)}`,
        });
        if (resp.ok) {
          message.success(`已切换到用户 ${user.username}`);
          window.location.href = '/';
        } else {
          message.error('切换用户失败');
        }
      } catch (error) {
        message.error('切换用户失败');
      }
    },
  });
}

async function handleSave() {
  saving.value = true;
  try {
    if (editingUser.value) {
      await updateUser(editingUser.value.id, {
        username: formData.value.username,
        role: formData.value.role,
        account_locked: formData.value.account_locked,
        password_change_required: formData.value.password_change_required,
      });
      message.success('用户更新成功');
    } else {
      await createUser(formData.value);
      message.success('用户创建成功');
    }
    modalVisible.value = false;
    loadUsers();
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '保存失败');
  } finally {
    saving.value = false;
  }
}

function handleDelete(user: any) {
  if (user.username === 'admin') {
    Modal.info({
      title: '无法删除用户',
      content: `用户账户 ${user.username} 无法被删除。`,
    });
    return;
  }
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除用户 ${user.username} 吗？其所有关联对象也将被删除，此操作不可恢复。`,
    onOk: async () => {
      try {
        await deleteUser(user.id);
        message.success('删除成功');
        loadUsers();
      } catch (error) {
        message.error('删除失败');
      }
    },
  });
}
</script>

<style scoped>
.users-container {
  padding: 24px;
}
</style>
<template>
  <div class="groups-container">
    <a-card>
      <template #extra>
        <a-space>
          <a-button type="primary" @click="showCreateModal">
            <PlusOutlined /> 新建用户组
          </a-button>
        </a-space>
      </template>
      
      <a-table
        :columns="columns"
        :data-source="groups"
        :loading="loading"
        :pagination="{ pageSize: 10 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a-tag color="cyan">{{ record.name }}</a-tag>
          </template>
          <template v-if="column.key === 'num_targets'">
            <a-tag :color="targetCountColor(record.targets?.length || 0)">{{ record.targets ? record.targets.length : 0 }}</a-tag>
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
      :title="editingGroup ? '编辑用户组' : '新建用户组'"
      @ok="handleSave"
      :confirm-loading="saving"
      width="800px"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="用户组名称" required>
          <a-input v-model:value="formData.name" placeholder="输入用户组名称" />
        </a-form-item>
        <a-form-item label="目标用户">
          <a-table
            :columns="targetColumns"
            :data-source="formData.targets"
            :pagination="false"
            size="small"
            row-key="email"
          >
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.key === 'email'">
                <a-input v-model:value="record.email" size="small" />
              </template>
              <template v-if="column.key === 'full_name'">
                <a-input v-model:value="record.full_name" size="small" />
              </template>
              <template v-if="column.key === 'position'">
                <a-input v-model:value="record.position" size="small" />
              </template>
              <template v-if="column.key === 'action'">
                <a-button size="small" danger @click="removeTarget(index)">
                  <DeleteOutlined />
                </a-button>
              </template>
            </template>
          </a-table>
          <a-space style="margin-top: 8px">
            <a-button @click="addTarget">
              <PlusOutlined /> 添加用户
            </a-button>
            <a-upload
              :before-upload="handleCsvUpload"
              :show-upload-list="false"
              accept=".csv"
            >
              <a-button>
                <ImportOutlined /> 导入CSV
              </a-button>
            </a-upload>
            <a-button @click="downloadCSVTemplate">
              <DownloadOutlined /> 下载CSV模板
            </a-button>
          </a-space>
          <div class="csv-format-hint">
            CSV格式：email, full_name, position
          </div>
        </a-form-item>
      </a-form>
    </a-modal>


  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  PlusOutlined,
  ImportOutlined,
  DeleteOutlined,
  DownloadOutlined,
} from '@ant-design/icons-vue';
import { getGroups, createGroup, updateGroup, deleteGroup, importGroup } from '@/api/groups';
import { formatDate } from '@/utils/format';

const loading = ref(false);
const saving = ref(false);
const groups = ref<any[]>([]);
const modalVisible = ref(false);
const editingGroup = ref<any>(null);

const formData = ref({
  name: '',
  targets: [] as any[],
});

const columns = [
  { title: '用户组名称', dataIndex: 'name', key: 'name' },
  { title: '用户数量', key: 'num_targets' },
  { title: '修改时间', dataIndex: 'modified_date', key: 'modified_date' },
  { title: '操作', key: 'action' },
];

const targetColumns = [
  { title: '邮箱', key: 'email', width: 200 },
  { title: '姓名', key: 'full_name', width: 200 },
  { title: '职位', key: 'position', width: 150 },
  { title: '操作', key: 'action', width: 60 },
];

onMounted(() => {
  loadGroups();
});

async function loadGroups() {
  loading.value = true;
  try {
    groups.value = (await getGroups()).sort(
      (a: any, b: any) => new Date(b.modified_date).getTime() - new Date(a.modified_date).getTime()
    );
  } catch (error) {
    message.error('加载用户组失败');
  } finally {
    loading.value = false;
  }
}

function showCreateModal() {
  editingGroup.value = null;
  formData.value = {
    name: '',
    targets: [],
  };
  modalVisible.value = true;
}

function showEditModal(group: any) {
  editingGroup.value = group;
  formData.value = {
    name: group.name,
    targets: group.targets ? [...group.targets] : [],
  };
  modalVisible.value = true;
}

function handleDuplicate(group: any) {
  editingGroup.value = null;
  formData.value = {
    name: `${group.name} (副本)`,
    targets: group.targets ? [...group.targets] : [],
  };
  modalVisible.value = true;
}

function addTarget() {
  formData.value.targets.push({
    email: '',
    full_name: '',
    position: '',
  });
}

function removeTarget(index: number) {
  formData.value.targets.splice(index, 1);
}

function downloadCSVTemplate() {
  const csvContent = 'email,full_name,position\r\nfoobar@example.com,Example User,Systems Administrator\r\n';
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.setAttribute('download', 'group_template.csv');
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

async function handleSave() {
  saving.value = true;
  try {
    if (editingGroup.value) {
      await updateGroup(editingGroup.value.id, formData.value);
      message.success('用户组更新成功');
    } else {
      await createGroup(formData.value);
      message.success('用户组创建成功');
    }
    modalVisible.value = false;
    loadGroups();
  } catch (error: any) {
    message.error(error?.message || '保存失败');
  } finally {
    saving.value = false;
  }
}

async function handleCsvUpload(file: File) {
  try {
    // Use backend API for robust CSV parsing
    const targets = await importGroup(0, file);
    if (Array.isArray(targets) && targets.length > 0) {
      formData.value.targets.push(...targets);
      message.success(`已导入 ${targets.length} 个用户`);
    } else {
      message.warning('CSV 文件为空或格式不正确');
    }
  } catch (error) {
    // Fallback to local parsing if backend import fails
    const reader = new FileReader();
    reader.onload = (e) => {
      const text = e.target?.result as string;
      const lines = text.split('\n').filter(line => line.trim());
      const startIndex = lines[0]?.toLowerCase().includes('email') ? 1 : 0;
      const newTargets: any[] = [];
      for (let i = startIndex; i < lines.length; i++) {
        const parts = lines[i].split(',').map(s => s.trim());
        if (parts[0]) {
          newTargets.push({
            email: parts[0] || '',
            full_name: parts[1] || '',
            position: parts[2] || '',
          });
        }
      }
      formData.value.targets.push(...newTargets);
      message.success(`已导入 ${newTargets.length} 个用户`);
    };
    reader.readAsText(file);
  }
  return false; // prevent default upload
}

function targetCountColor(count: number) {
  if (count === 0) return 'default';
  if (count < 10) return 'blue';
  if (count < 50) return 'geekblue';
  return 'purple';
}

function handleDelete(id: number) {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除这个用户组吗？此操作不可恢复。',
    onOk: async () => {
      try {
        await deleteGroup(id);
        message.success('删除成功');
        loadGroups();
      } catch (error) {
        message.error('删除失败');
      }
    },
  });
}
</script>

<style scoped>
.groups-container {
  padding: 24px;
}

.csv-format-hint {
  margin-top: 8px;
  color: #666;
  font-size: 12px;
}
</style>

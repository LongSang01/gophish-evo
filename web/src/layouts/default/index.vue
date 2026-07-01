<template>
  <a-layout class="default-layout">
    <a-layout-sider
      v-model:collapsed="collapsed"
      collapsible
      :trigger="null"
      breakpoint="lg"
      :width="220"
      theme="dark"
    >
      <div class="logo">
        <h1>Gophish-Evo</h1>
      </div>
      <a-menu
        v-model:selectedKeys="selectedKeys"
        v-model:openKeys="openKeys"
        theme="dark"
        mode="inline"
        @click="handleMenuClick"
      >
        <template v-for="route in menuRoutes" :key="route.path">
          <a-menu-item v-if="!route.children" :key="route.path">
            <component :is="$antIcons[route.meta?.icon as string]" v-if="route.meta?.icon" />
            <span>{{ route.meta?.title }}</span>
          </a-menu-item>
        </template>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <a-layout-header class="layout-header">
        <a-space>
          <MenuFoldOutlined
            v-if="!collapsed"
            class="trigger"
            @click="() => (collapsed = !collapsed)"
          />
          <MenuUnfoldOutlined
            v-else
            class="trigger"
            @click="() => (collapsed = !collapsed)"
          />
        </a-space>
        <a-space class="header-right">
          <a-dropdown>
            <a-space class="user-dropdown">
              <UserOutlined />
              <span>{{ userStore.userInfo?.username || '用户' }}</span>
            </a-space>
            <template #overlay>
              <a-menu @click="handleUserMenuClick">
                <a-menu-item key="settings">
                  <SettingOutlined />
                  <span style="margin-left: 8px">设置</span>
                </a-menu-item>
                <a-menu-divider />
                <a-menu-item key="logout">
                  <LogoutOutlined />
                  <span style="margin-left: 8px">退出登录</span>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </a-space>
      </a-layout-header>
      <a-layout-content class="layout-content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { message } from 'ant-design-vue';
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  DashboardOutlined,
  MailOutlined,
  FileTextOutlined,
  TeamOutlined,
  GlobalOutlined,
  SendOutlined,
  ApiOutlined,
} from '@ant-design/icons-vue';
import { useUserStore } from '@/store/modules/user';

const router = useRouter();
const route = useRoute();
const userStore = useUserStore();
const collapsed = ref(false);
const selectedKeys = ref<string[]>([]);
const openKeys = ref<string[]>([]);

// Register icons globally for dynamic use
const $antIcons: Record<string, any> = {
  DashboardOutlined,
  MailOutlined,
  FileTextOutlined,
  TeamOutlined,
  GlobalOutlined,
  SendOutlined,
  ApiOutlined,
  UserOutlined,
  SettingOutlined,
};

const menuRoutes = computed(() => {
  const mainRoute = router.options.routes.find(r => r.path === '/');
  if (!mainRoute?.children) return [];
  return mainRoute.children.filter(r => !r.meta?.hideMenu && r.meta?.title);
});

watch(
  () => route.path,
  (path) => {
    selectedKeys.value = [path.startsWith('/') ? path.slice(1) : path];
  },
  { immediate: true }
);

function handleMenuClick({ key }: { key: string }) {
  router.push('/' + key);
}

async function handleUserMenuClick({ key }: { key: string }) {
  switch (key) {
    case 'settings':
      router.push('/settings');
      break;
    case 'logout':
      try {
        await userStore.logoutAction();
        message.success('已退出登录');
        router.push('/login');
      } catch (error) {
        router.push('/login');
      }
      break;
  }
}
</script>

<style scoped>
.default-layout {
  min-height: 100vh;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.logo h1 {
  margin: 0;
  color: #fff;
  font-size: 20px;
}

.layout-header {
  background: #fff;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.trigger {
  font-size: 18px;
  cursor: pointer;
  transition: color 0.3s;
}

.trigger:hover {
  color: #1890ff;
}

.header-right {
  display: flex;
  align-items: center;
}

.user-dropdown {
  cursor: pointer;
  padding: 0 12px;
}

.layout-content {
  margin: 0;
  padding: 24px;
  background: #f5f7fa;
  min-height: calc(100vh - 64px);
}

.layout-content :deep(.ant-card) {
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.06);
  transition: box-shadow 0.3s;
}

.layout-content :deep(.ant-card:hover) {
  box-shadow: 0 2px 12px rgba(0,0,0,0.1);
}
</style>

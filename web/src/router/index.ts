import { createRouter, createWebHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';
import { useUserStore } from '@/store/modules/user';

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录', requiresAuth: false },
  },
  {
    path: '/',
    component: () => import('@/layouts/default/index.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '仪表盘', icon: 'DashboardOutlined' },
      },
      {
        path: 'campaigns',
        name: 'Campaigns',
        component: () => import('@/views/campaigns/index.vue'),
        meta: { title: '钓鱼活动', icon: 'MailOutlined' },
      },
      {
        path: 'campaigns/:id',
        name: 'CampaignDetail',
        component: () => import('@/views/campaigns/detail.vue'),
        meta: { title: '活动详情', hideMenu: true },
      },
      {
        path: 'templates',
        name: 'Templates',
        component: () => import('@/views/templates/index.vue'),
        meta: { title: '邮件模板', icon: 'FileTextOutlined' },
      },
      {
        path: 'groups',
        name: 'Groups',
        component: () => import('@/views/groups/index.vue'),
        meta: { title: '用户组', icon: 'TeamOutlined' },
      },
      {
        path: 'pages',
        name: 'Pages',
        component: () => import('@/views/pages/index.vue'),
        meta: { title: '落地页', icon: 'GlobalOutlined' },
      },
      {
        path: 'smtp',
        name: 'SMTP',
        component: () => import('@/views/smtp/index.vue'),
        meta: { title: '发送配置', icon: 'SendOutlined' },
      },
      {
        path: 'webhooks',
        name: 'Webhooks',
        component: () => import('@/views/webhooks/index.vue'),
        meta: { title: 'Webhooks', icon: 'ApiOutlined' },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/users/index.vue'),
        meta: { title: '用户管理', icon: 'UserOutlined', requiresAdmin: true },
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/settings/index.vue'),
        meta: { title: '设置', icon: 'SettingOutlined' },
      },
    ],
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

// Navigation guard
router.beforeEach(async (to, _from, next) => {
  const userStore = useUserStore();

  // Set page title
  document.title = `${to.meta.title || 'Gophish-Evo'} - Gophish-Evo Admin`;

  // Check if route requires authentication
  if (to.meta.requiresAuth !== false) {
    // Check if user is logged in
    if (!userStore.token) {
      next({ name: 'Login', query: { redirect: to.fullPath } });
      return;
    }

    // Check if user info is loaded
    if (!userStore.userInfo) {
      try {
        await userStore.getUserInfo();
      } catch (error) {
        next({ name: 'Login', query: { redirect: to.fullPath } });
        return;
      }
    }

    // Check admin permission
    if (to.meta.requiresAdmin && userStore.userInfo?.role?.slug !== 'admin') {
      next({ name: 'Dashboard' });
      return;
    }
  }

  next();
});

export default router;

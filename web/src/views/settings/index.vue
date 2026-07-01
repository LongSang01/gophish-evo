<template>
  <div class="settings-container">
    <a-row :gutter="24">
      <a-col :span="12">
        <a-card title="修改密码">
          <a-form :model="passwordForm" layout="vertical" @finish="handleChangePassword">
            <a-form-item label="当前密码" required>
              <a-input-password v-model:value="passwordForm.current_password" placeholder="输入当前密码" />
            </a-form-item>
            <a-form-item label="新密码" required>
              <a-input-password v-model:value="passwordForm.new_password" placeholder="输入新密码" />
            </a-form-item>
            <a-form-item label="确认新密码" required>
              <a-input-password v-model:value="passwordForm.confirm_password" placeholder="再次输入新密码" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" html-type="submit" :loading="changingPassword">
                修改密码
              </a-button>
            </a-form-item>
          </a-form>
        </a-card>

        <a-card title="IMAP设置" style="margin-top: 24px">
          <a-form :model="imapForm" layout="vertical">
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item>
                  <a-checkbox v-model:checked="imapForm.enabled">启用IMAP</a-checkbox>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="服务器地址" required>
                  <a-input v-model:value="imapForm.host" placeholder="imap.example.com" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="端口" required>
                  <a-input-number v-model:value="imapForm.port" :min="1" :max="65535" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="用户名" required>
                  <a-input v-model:value="imapForm.username" placeholder="IMAP用户名" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="密码">
                  <a-input-password v-model:value="imapForm.password" placeholder="IMAP密码" />
                </a-form-item>
              </a-col>
              <a-col :span="24">
                <a-divider style="margin: 4px 0 16px" />
              </a-col>
              <a-col :span="12">
                <a-form-item>
                  <a-checkbox v-model:checked="imapForm.tls">使用TLS</a-checkbox>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item>
                  <a-checkbox v-model:checked="imapForm.ignore_cert_errors">忽略证书错误</a-checkbox>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="邮箱文件夹">
                  <a-input v-model:value="imapForm.folder" placeholder="INBOX" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="轮询频率（秒）">
                  <a-input-number v-model:value="imapForm.imap_freq" :min="10" :max="3600" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="24">
                <a-form-item label="限制域名">
                  <a-input v-model:value="imapForm.restrict_domain" placeholder="仅处理来自此域名的邮件（可选）" />
                </a-form-item>
              </a-col>
              <a-col :span="24">
                <a-form-item>
                  <a-checkbox v-model:checked="imapForm.delete_reported_campaign_email">删除已上报的钓鱼邮件</a-checkbox>
                </a-form-item>
              </a-col>
              <a-col :span="24">
                <a-form-item>
                  <a-space>
                    <a-button type="primary" @click="handleSaveIMAP" :loading="savingIMAP">保存设置</a-button>
                    <a-button @click="handleValidateIMAP" :loading="validatingIMAP">测试连接</a-button>
                  </a-space>
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="系统信息">
          <a-descriptions :column="1" bordered>
            <a-descriptions-item label="版本">{{ systemInfo.version }}</a-descriptions-item>
            <a-descriptions-item label="构建时间">{{ systemInfo.buildTime }}</a-descriptions-item>
            <a-descriptions-item label="API版本">{{ systemInfo.apiVersion }}</a-descriptions-item>
          </a-descriptions>
        </a-card>

        <a-card title="API密钥" style="margin-top: 24px">
          <a-alert
            message="API密钥用于程序化访问Gophish-Evo API"
            type="info"
            show-icon
            style="margin-bottom: 16px"
          />
          <a-space>
            <a-input-password
              :value="apiKey"
              readonly
              style="width: 400px"
              placeholder="未设置"
            />
            <a-button @click="copyApiKey">复制</a-button>
            <a-button @click="regenerateApiKey" danger>重新生成</a-button>
          </a-space>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { changePassword, getCurrentUser, resetApiKey } from '@/api/auth';
import { getIMAPSettings, saveIMAPSettings, validateIMAP } from '@/api/imap';
import { useUserStore } from '@/store/modules/user';

const userStore = useUserStore();

const changingPassword = ref(false);
const savingIMAP = ref(false);
const validatingIMAP = ref(false);
const apiKey = ref('');
const systemInfo = ref({
  version: '0.12.0',
  buildTime: '-',
  apiVersion: 'v1',
});

const passwordForm = ref({
  current_password: '',
  new_password: '',
  confirm_password: '',
});

const imapForm = ref({
  enabled: false,
  host: '',
  port: 993,
  username: '',
  password: '',
  tls: true,
  ignore_cert_errors: false,
  folder: 'INBOX',
  imap_freq: 60,
  restrict_domain: '',
  delete_reported_campaign_email: false,
});

onMounted(() => {
  loadSystemInfo();
  loadApiKey();
  loadIMAPSettings();
});

async function loadSystemInfo() {
  try {
    const versionModule = await import('@/../../VERSION?raw');
    const version = versionModule.default?.trim() || '1.1.0';
    systemInfo.value = {
      version,
      buildTime: new Date().toISOString(),
      apiVersion: 'v1',
    };
  } catch (error) {
    systemInfo.value = {
      version: '1.1.0',
      buildTime: '-',
      apiVersion: 'v1',
    };
  }
}

async function loadApiKey() {
  try {
    let user = userStore.userInfo;
    if (!user) {
      user = await getCurrentUser();
    }
    apiKey.value = user?.api_key || '未设置';
  } catch (error) {
    apiKey.value = '获取失败';
  }
}

async function loadIMAPSettings() {
  try {
    const settings = await getIMAPSettings();
    if (settings && settings.length > 0) {
      const s = settings[0];
      imapForm.value = {
        enabled: s.enabled || false,
        host: s.host || '',
        port: s.port || 993,
        username: s.username || '',
        password: s.password || '',
        tls: s.tls !== false,
        ignore_cert_errors: s.ignore_cert_errors || false,
        folder: s.folder || 'INBOX',
        imap_freq: s.imap_freq || 60,
        restrict_domain: s.restrict_domain || '',
        delete_reported_campaign_email: s.delete_reported_campaign_email || false,
      };
    }
  } catch (error) {
    // silent
  }
}

async function handleChangePassword() {
  if (passwordForm.value.new_password !== passwordForm.value.confirm_password) {
    message.error('两次输入的密码不一致');
    return;
  }
  if (!passwordForm.value.new_password) {
    message.error('请输入新密码');
    return;
  }
  changingPassword.value = true;
  try {
    await changePassword({
      current_password: passwordForm.value.current_password,
      new_password: passwordForm.value.new_password,
      confirm_password: passwordForm.value.confirm_password,
    });
    message.success('密码修改成功');
    passwordForm.value = {
      current_password: '',
      new_password: '',
      confirm_password: '',
    };
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '密码修改失败');
  } finally {
    changingPassword.value = false;
  }
}

async function handleSaveIMAP() {
  if (!imapForm.value.host) {
    message.error('请输入IMAP服务器地址');
    return;
  }
  savingIMAP.value = true;
  try {
    await saveIMAPSettings(imapForm.value);
    message.success('IMAP设置保存成功');
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '保存失败');
  } finally {
    savingIMAP.value = false;
  }
}

async function handleValidateIMAP() {
  if (!imapForm.value.host) {
    message.error('请输入IMAP服务器地址');
    return;
  }
  validatingIMAP.value = true;
  try {
    await validateIMAP({
      host: imapForm.value.host,
      port: imapForm.value.port,
      username: imapForm.value.username,
      password: imapForm.value.password,
      tls: imapForm.value.tls,
      ignore_cert_errors: imapForm.value.ignore_cert_errors,
    });
    message.success('IMAP连接测试成功');
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '连接测试失败');
  } finally {
    validatingIMAP.value = false;
  }
}

function copyApiKey() {
  if (apiKey.value) {
    navigator.clipboard.writeText(apiKey.value);
    message.success('API密钥已复制到剪贴板');
  }
}

function regenerateApiKey() {
  Modal.confirm({
    title: '确认重新生成',
    content: '重新生成API密钥后，旧密钥将立即失效。确定继续吗？',
    onOk: async () => {
      try {
        const res = await resetApiKey();
        if (res?.data) {
          apiKey.value = res.data;
        } else {
          await loadApiKey();
        }
        message.success('API密钥已重新生成');
      } catch (error) {
        message.error('重新生成失败');
      }
    },
  });
}
</script>

<style scoped>
.settings-container {
  padding: 24px;
}
</style>
import type { App } from 'vue';
import Antd from 'ant-design-vue';

export function setupAntd(app: App) {
  app.use(Antd);
}

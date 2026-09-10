import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import router from './router';
import { setupAntd } from './plugins/antd';

import 'ant-design-vue/dist/reset.css';
import './styles/index.css';

const app = createApp(App);

// Setup plugins
app.use(createPinia());
app.use(router);
app.use(setupAntd);

app.mount('#app');
